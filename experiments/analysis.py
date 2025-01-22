import pandas as pd
import seaborn as sns
import matplotlib.pyplot as plt
from pathlib import Path
import glob
import numpy as np
from typing import List, Dict, Generator
import random
import matplotlib.animation as animation
import math


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


def plot_episodes_violin(move_records, agent_configs, output_dir):
    """Create a violin plot of episode distributions for each concurrency level."""
    plt.figure(figsize=(12, 6))

    # Create violin plot
    sns.violinplot(data=move_records, x="goroutines", y="episodes", inner="box", cut=0)

    plt.title("Distribution of Search Episodes by Concurrency Level")
    plt.xlabel("Number of Goroutines")
    plt.ylabel("Search Episodes")

    plt.savefig(Path(output_dir) / "episodes_violin.png")
    return plt

def plot_episodes_by_cutoff(move_records, agent_configs, output_dir):
    """Create a violin plot of episode distributions for each cutoff depth."""
    plt.figure(figsize=(12, 6))

    # Create violin plot
    sns.boxplot(data=move_records, x="cutoff", y="episodes")

    plt.title("Distribution of Search Episodes by Cutoff Depth")
    plt.xlabel("Cutoff Depth")
    plt.ylabel("Search Episodes")

    plt.savefig(Path(output_dir) / "episodes_violin_cutoff.png")
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

    results.columns = ["agent2", "games", "wins"]
    results["win_rate"] = results["wins"] / results["games"]

    # Merge with agent configs to get configuration parameters
    results = results.merge(agent_configs, left_on="agent2", right_on="id", how="left")

    return results


def process_cutoff_labels(win_rates: pd.DataFrame) -> pd.DataFrame:
    """Process cutoff depth labels by adding descriptive postfixes.

    Args:
        win_rates: DataFrame containing win rates with 'cutoff' column

    Returns:
        DataFrame with processed cutoff labels
    """
    # Order by cutoff depth for plotting
    win_rates = win_rates.sort_values("cutoff")

    # Get no cutoff (full playout) row
    full_playout = win_rates.iloc[0:1].copy()
    full_playout["cutoff"] = "Full Playout (No Cutoff)"

    # Process remaining cutoff depth rows
    cutoff_depths = win_rates.iloc[1:].copy()
    label_postfixes = [
        "(Lower Fence)",
        "(Lower Quartile)",
        "(Median)",
        "(Upper Quartile)",
        # "(Upper Fence)",
    ]
    cutoff_depths["cutoff"] = [
        f"{depth} {label_postfixes[i]}"
        for i, depth in enumerate(cutoff_depths["cutoff"])
    ]

    # Combine and return results
    return pd.concat([cutoff_depths, full_playout])


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


def plot_combined_win_rates(
    win_rates_list, param_col, output_dir, title=None, xlabel=None
):
    """Plot win rates from multiple experiments on the same chart.

    Args:
        win_rates_list: List of (win_rates DataFrame, label) tuples
        param_col: Column name for x-axis values
        output_dir: Directory to save plot
        title: Optional plot title
        xlabel: Optional x-axis label
    """
    plt.figure(figsize=(10, 6))

    for win_rates, label in win_rates_list:
        x_positions = np.arange(len(win_rates))

        plt.plot(
            x_positions,
            win_rates["win_rate"],
            marker="o",
            linestyle="-",
            linewidth=2,
            label=label,
        )

    # Set title and labels
    plt.title(title or "Win Rates Comparison")
    plt.xlabel(xlabel or param_col.title())
    plt.ylabel("Win Rate Against Baseline Agent")

    # Set x-axis ticks and labels using first dataset
    x_labels = win_rates_list[0][0][param_col]
    plt.xticks(np.arange(len(x_labels)), x_labels)

    # Add 50% line to show baseline performance
    plt.axhline(y=0.5, color="r", linestyle="--", alpha=0.5, label="Baseline")

    plt.grid(True)
    plt.legend()
    plt.savefig(Path(output_dir) / "combined_win_rates.png")
    return plt


def calculate_k_factor(num_games: int) -> float:
    """Calculate K-factor with exponential decay from 32 to 16.
    
    Args:
        num_games: Number of games played by agent
        
    Returns:
        K-factor between 32 (initial) and 16 (minimum)
    """
    K_START = 32.0
    K_END = 16.0
    # Decay constant controls how quickly K approaches K_END
    # With -0.01, reaches ~95% of decay after 300 games
    DECAY_RATE = -0.01
    
    decay = math.exp(DECAY_RATE * num_games)
    k_factor = K_END + (K_START - K_END) * decay
    return k_factor


def calculate_elo_updates(game_records: pd.DataFrame, seed: int = 42) -> pd.DataFrame:
    """Calculate Elo rating updates with exponentially decaying K-factor."""
    # Initialize ratings at 1500
    ratings = {agent_id: 1500.0 for agent_id in pd.unique(game_records[['agent1', 'agent2']].values.ravel())}
    ratings_history = []
    games_played = {agent_id: 0 for agent_id in ratings.keys()}
    
    # Process each game in random order - sample ONCE before the loop
    randomized_games = game_records.sample(frac=1, random_state=seed)  # Fixed seed for reproducibility
    
    for _, game in randomized_games.iterrows():
        # Get current ratings
        rating1 = ratings[game.agent1]
        rating2 = ratings[game.agent2]
        
        # Calculate expected scores
        expected1 = 1 / (1 + 10**((rating2 - rating1) / 400))
        expected2 = 1 - expected1
        
        # Get actual scores (1 for win, 0 for loss)
        actual1 = 1 if game.winner == "Player1" else 0
        actual2 = 1 - actual1
        
        # Calculate K-factors based on games played
        k1 = calculate_k_factor(games_played[game.agent1])
        k2 = calculate_k_factor(games_played[game.agent2])
        
        # Update ratings
        ratings[game.agent1] += k1 * (actual1 - expected1)
        ratings[game.agent2] += k2 * (actual2 - expected2)
        
        # Track games played
        games_played[game.agent1] += 1
        games_played[game.agent2] += 1
        
        # Store ratings snapshot
        ratings_history.append(ratings.copy())
    
    # Convert history to DataFrame
    return pd.DataFrame(ratings_history)


def plot_elo_progression(ratings_df: pd.DataFrame, agent_configs: pd.DataFrame, output_dir: Path):
    """Create a static line plot showing Elo rating progression over time.
    
    Args:
        ratings_df: DataFrame containing Elo rating history
        agent_configs: DataFrame containing agent configurations
        output_dir: Directory to save the plot
    """
    # Set up the figure
    plt.figure(figsize=(10, 6))
    
    # Plot line for each agent
    for agent in agent_configs['id']:
        plt.plot(
            range(len(ratings_df)), 
            ratings_df[agent],
            label=f"Agent {agent}",
            linewidth=2
        )
    
    # Set title and labels
    plt.title("Elo Rating Progression")
    plt.xlabel("Games Played")
    plt.ylabel("Elo Rating")
    
    # Add baseline rating line
    plt.axhline(y=1500, color='r', linestyle='--', alpha=0.5, label='Initial Rating')
    
    plt.grid(True)
    plt.legend()
    
    # Save plot
    plt.savefig(Path(output_dir) / 'elo_progression.png')
    return plt


def plot_final_elo_ratings(final_ratings: pd.DataFrame, agent_configs: pd.DataFrame, output_dir: Path):
    """Create a line plot of final Elo ratings.
    
    Args:
        final_ratings: DataFrame containing final Elo ratings
        agent_configs: DataFrame containing agent configurations
        output_dir: Directory to save the plot
    """
    # Create line plot
    plt.figure(figsize=(10, 6))
    
    agents = sorted(final_ratings.index)
    ratings = [final_ratings[agent] for agent in agents]
    
    # Plot line with markers
    plt.plot(agents, ratings, marker='o', linestyle='-', linewidth=2, markersize=8)
    
    # Add baseline rating line
    plt.axhline(y=1500, color='r', linestyle='--', alpha=0.5, label='Initial Rating')
    
    plt.title("Final Elo Ratings")
    plt.xlabel("Agent ID")
    plt.ylabel("Elo Rating")
    plt.grid(True)
    plt.legend()
    
    # Set x-axis ticks to agent IDs
    plt.xticks(agents)
    
    plt.savefig(Path(output_dir) / 'final_elo_ratings.png')
    return plt


def calculate_pairwise_win_rates(game_records: pd.DataFrame, agent_configs: pd.DataFrame) -> pd.DataFrame:
    """Calculate win rates between each pair of agents.
    
    Args:
        game_records: DataFrame containing game results
        agent_configs: DataFrame containing agent configurations
        
    Returns:
        DataFrame with win rates from row agent's perspective against column agent
    """
    agents = sorted(agent_configs['id'].unique())
    win_rates = pd.DataFrame(np.nan, index=agents, columns=agents)  # Initialize with NaN

    # Calculate win rate for each pair
    for agent1 in agents:
        for agent2 in agents:
            if agent1 != agent2:  # Skip diagonal entries
                # Get games between these agents
                mask = ((game_records['agent1'] == agent1) & (game_records['agent2'] == agent2)) | \
                       ((game_records['agent1'] == agent2) & (game_records['agent2'] == agent1))
                games = game_records[mask]
                
                if len(games) > 0:
                    # Calculate win rate from agent1's perspective
                    wins = sum(
                      ((games['agent1'] == agent1) &
                      (games['winner'] == 'Player1')) |
                      ((games['agent2'] == agent1) &
                       (games['winner'] == 'Player2'))
                    )
                    win_rates.loc[agent1, agent2] = wins / len(games)
                    # print number of wins and total games 
                    print(f"Agent {agent1} vs Agent {agent2}: {wins} wins out of {len(games)} games")
    
    return win_rates


def plot_win_rate_heatmap(win_rates: pd.DataFrame, agent_configs: pd.DataFrame, output_dir: Path):
    """Create a heatmap showing win rates between all agent pairs.
    
    Args:
        win_rates: DataFrame containing pairwise win rates
        agent_configs: DataFrame containing agent configurations
        output_dir: Directory to save the plot
    """
    plt.figure(figsize=(10, 8))
    
    # Create mask for diagonal entries
    mask = np.zeros_like(win_rates)
    np.fill_diagonal(mask, True)
    
    # Create heatmap
    sns.heatmap(
        win_rates,
        annot=True,  # Show values in cells
        fmt='.2f',   # Format as 2 decimal places
        cmap='RdYlBu',  # Red (low) to Blue (high)
        center=0.5,  # Center colormap at 0.5
        vmin=0,      # Minimum value
        vmax=1,      # Maximum value
        square=True, # Make cells square
        mask=mask,   # Mask diagonal entries
        cbar_kws={'label': 'Win Rate (Row Agent)'},
        annot_kws={'va': 'center'}  # Center annotations vertically
    )
    
    # Add "N/A" text in diagonal entries
    for i in range(len(win_rates)):
        plt.text(i + 0.5, i + 0.5, 'N/A', 
                horizontalalignment='center',
                verticalalignment='center')
    
    plt.title("Pairwise Win Rates\n(Row Agent vs Column Agent)")
    plt.xlabel("Opponent Agent ID")
    plt.ylabel("Agent ID")
    
    plt.tight_layout()
    plt.savefig(Path(output_dir) / 'win_rate_heatmap.png')
    return plt

def show_missing_data(game_records: pd.DataFrame):
    return game_records[game_records['winner'].isna()]