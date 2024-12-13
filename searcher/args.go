package searcher

// Hyperparameters for MCTS

const C_SQUARED = 2.0

// Use rewards to estimate the chance of winning
const WIN = 1.0
const LOSS = 1 - WIN

// TODO: AlphaZero uses rewards to estimate the expected outcome
