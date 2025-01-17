import pandas as pd
import seaborn as sns
import matplotlib.pyplot as plt
from pathlib import Path
import glob
import numpy as np
from typing import List, Dict, Generator
import random
import matplotlib.animation as animation


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

    results.columns = ["agent2", "games", "wins"]  # TODO: rename during agg()
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
        "",
        "(Lower Quartile)",
        "(Median)",
        "(Upper Quartile)",
        "(Upper Fence)",
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
