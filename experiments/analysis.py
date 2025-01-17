import pandas as pd
import seaborn as sns
import matplotlib.pyplot as plt
from pathlib import Path
import glob
import numpy as np
from typing import List, Dict


def load_experiment_data(experiment_dir):
    """Load the latest experiment data from CSV files."""
    # Get the latest experiment timestamp directory
    timestamp_dirs = sorted(glob.glob(f"{experiment_dir}/*/"))
    if not timestamp_dirs:
        raise ValueError(f"No experiment data found in {experiment_dir}")
    latest_dir = timestamp_dirs[-1]

    # Load the CSV files
    agent_configs = pd.read_csv(Path(latest_dir) / "agent_configs.csv")
    game_records = pd.read_csv(Path(latest_dir) / "game_records.csv")
    move_records = pd.read_csv(Path(latest_dir) / "move_records.csv")

    # Create output directory to store plots
    output_dir = Path(latest_dir) / "output"
    output_dir.mkdir(parents=True, exist_ok=True)

    return agent_configs, game_records, move_records, output_dir


def plot_episodes_by_step(move_records, agent_configs, output_dir):
    """Create scatter plots of episodes vs step for each concurrency level."""
    plt.figure(figsize=(15, 10))

    # Create subplot for each concurrency level
    unique_goroutines = agent_configs["goroutines"].unique()
    num_plots = len(unique_goroutines)
    rows = (num_plots + 2) // 3  # Ceiling division
    cols = min(3, num_plots)

    # Create figure with shared axes
    fig, axes = plt.subplots(rows, cols, figsize=(15, 10), sharex=True, sharey=True)
    axes = axes.ravel() if num_plots > 1 else [axes]

    for i, goroutines in enumerate(sorted(unique_goroutines)):
        # Filter moves for this concurrency level
        moves = move_records[move_records["goroutines"] == goroutines]

        # Plot episodes vs step
        sns.scatterplot(data=moves, x="step", y="episodes", alpha=0.5, ax=axes[i])

        axes[i].set_title(f"Goroutines: {goroutines}")
        axes[i].set_xlabel("Move Step")
        axes[i].set_ylabel("Search Episodes")

    # Hide empty subplots if any
    for j in range(i + 1, len(axes)):
        axes[j].set_visible(False)

    plt.tight_layout()
    plt.savefig(Path(output_dir) / "episodes_by_step.png")
    return plt


def plot_episodes_histogram(move_records, agent_configs, output_dir):
    """Create histograms of episode counts for each concurrency level."""
    plt.figure(figsize=(15, 10))

    unique_goroutines = agent_configs["goroutines"].unique()
    num_plots = len(unique_goroutines)
    rows = (num_plots + 2) // 3
    cols = min(3, num_plots)

    # Create figure with shared axes
    fig, axes = plt.subplots(rows, cols, figsize=(15, 10), sharex=True, sharey=True)
    axes = axes.ravel() if num_plots > 1 else [axes]

    for i, goroutines in enumerate(sorted(unique_goroutines)):
        moves = move_records[move_records["goroutines"] == goroutines]

        sns.histplot(data=moves, x="episodes", bins=30, ax=axes[i])

        axes[i].set_title(f"Goroutines: {goroutines}")
        axes[i].set_xlabel("Search Episodes")
        axes[i].set_ylabel("Count")

    # Hide empty subplots if any
    for j in range(i + 1, len(axes)):
        axes[j].set_visible(False)

    plt.tight_layout()
    plt.savefig(Path(output_dir) / "episodes_histogram.png")
    return plt


def plot_episodes_boxplot(move_records, agent_configs, output_dir):
    """Create a box plot of episode distributions for each concurrency level."""
    plt.figure(figsize=(12, 6))

    # Create box plot
    sns.boxplot(data=move_records, x="goroutines", y="episodes")

    plt.title("Distribution of Search Episodes by Concurrency Level")
    plt.xlabel("Number of Goroutines")
    plt.ylabel("Search Episodes")

    plt.savefig(Path(output_dir) / "episodes_boxplot.png")
    return plt


def calculate_win_rates(
    game_records: pd.DataFrame, agent_configs: pd.DataFrame
) -> pd.DataFrame:
    """Calculate win rates for each experiment agent against the baseline agent."""
    # Group by the experiment agents of different configuration parameters
    results = (
        game_records.groupby("agent2")
        .agg(
            {
                "id": "count",  # Total games
                "winner": lambda x: sum(x == "Player2"),  # Wins by experiment agent 
            }
        )
        .reset_index()
    )

    results.columns = ["agent2", "games", "wins"] # TODO: rename during agg()
    results["win_rate"] = results["wins"] / results["games"]

    # Merge with agent configs to get configuration parameters
    results = results.merge(
        agent_configs,
        left_on="agent2",
        right_on="id",
        how="left"
    )
    
    return results


def plot_win_rates(
    win_rates: pd.DataFrame, 
    param_col: str, 
    output_dir: Path,
    title: str = None,  
    xlabel: str = None,  
) -> plt.Figure:
    """Plot win rates against the varying configuration parameter.
    
    Args:
        win_rates: DataFrame containing win rates and configuration parameters
        param_col: Name of column containing the configuration parameter
        output_dir: Directory to save plot
        title: Optional custom title (default: "Win Rate vs {param_col}")
        xlabel: Optional custom x-axis label (default: param_col)
    """
    plt.figure(figsize=(10, 6))

    # Create evenly spaced x-axis positions
    x_positions = np.arange(len(win_rates))
    
    plt.plot(
        x_positions,
        win_rates["win_rate"],
        marker="o",
        linestyle="-", 
        linewidth=2,
    )

    # Set title and labels
    plt.title(title or f"Win Rate vs {param_col.title()}")
    plt.xlabel(xlabel or param_col.title())
    plt.ylabel("Win Rate Against Baseline Agent")

    # Set x-axis ticks and labels
    x_labels = win_rates[param_col]
    plt.xticks(x_positions, x_labels)

    # Add 50% line to show baseline performance
    plt.axhline(y=0.5, color="r", linestyle="--", alpha=0.5)

    plt.grid(True)
    plt.legend()
    plt.savefig(Path(output_dir) / "win_rates.png")
    return plt


def calculate_elo_ratings(
    game_records: pd.DataFrame, K: float = 16.0
) -> Dict[int, float]:
    """Calculate Elo ratings for all agents after playing all games.

    Args:
        game_records: DataFrame containing game results
        K: Elo K-factor (usually 16.0, 24.0, 32.0 or 40.0)

    Returns:
        Dictionary mapping agent IDs to their final Elo ratings
    """
    # TODO: initialize at 1200?
    # Initialize ratings for all agents (including baseline) at 1500
    ratings = {
        agent_id: 1500.0
        for agent_id in pd.concat(
            [game_records["agent1"], game_records["agent2"]]
        ).unique()
    }

    # Shuffle games randomly to minimize ordering effects on final ratings
    shuffled_games = game_records.sample(
        frac=1.0, 
        # random_state=12 # TODO fix random_state for reproducibility
    )

    # Update ratings by game outcomes in random order
    for _, game in shuffled_games.iterrows():
        agent1_id = game["agent1"]
        agent2_id = game["agent2"]

        # Get expected scores
        rating_diff = (ratings[agent2_id] - ratings[agent1_id]) / 400.0
        expected_1 = 1.0 / (1.0 + 10.0**rating_diff)
        expected_2 = 1.0 - expected_1

        # Get actual scores
        actual_1 = 1.0 if game["winner"] == "Player1" else 0.0
        actual_2 = 1.0 - actual_1

        # Update ratings
        ratings[agent1_id] += K * (actual_1 - expected_1)
        ratings[agent2_id] += K * (actual_2 - expected_2)

    return ratings


def plot_elo_ratings(ratings: Dict[int, float], agent_configs: pd.DataFrame, output_dir: Path) -> plt.Figure:
    """Plot Elo ratings against concurrency level."""
    plt.figure(figsize=(10, 6))

    # Convert ratings to DataFrame for plotting
    ratings_df = pd.DataFrame(
        [{"id": agent_id, "rating": rating} for agent_id, rating in ratings.items()]
    )

    # Merge with agent configs to get goroutines info
    ratings_df = ratings_df.merge(
        agent_configs[["id", "goroutines"]], on="id", how="left"
    )

    # Get baseline agent's rating (agent with ID 0)
    baseline_rating = ratings[0]

    # Sort by goroutines for line plot and exclude baseline agent
    ratings_df = ratings_df[ratings_df["id"] != 0].sort_values("goroutines")

    plt.plot(
        range(len(ratings_df)),  # Use evenly spaced x labels
        ratings_df["rating"],
        marker="o",
        linestyle="-",
        linewidth=2,
        # label="Parallel Agents"
    )

    plt.title("Elo Rating vs Concurrency Level")
    plt.xlabel("Number of Goroutines")
    plt.ylabel("Elo Rating")

    # Set x-axis ticks to show actual goroutine values as integers
    plt.xticks(range(len(ratings_df)), ratings_df["goroutines"])

    # Add initial rating line
    plt.axhline(y=1500, color="gray", linestyle="--", alpha=0.5, label="Initial Rating")

    # Add baseline rating line
    # plt.axhline(y=baseline_rating, color="r", linestyle="--", alpha=0.5, label="Baseline Agent")

    plt.legend()
    plt.grid(True)
    plt.savefig(Path(output_dir) / "elo_ratings.png")
    return plt
