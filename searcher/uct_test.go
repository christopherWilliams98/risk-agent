package searcher

// TODO: test chance nodes for stochastic moves
/* spec:
- selection:
	- happy path: fully expanded node with stochastic moves (decision parent chance node children) -> unvisited child or max UCB child
	- happy path: stochastic move node (chance node parent decision children), known outcome -> child node
	- otherwise: skip
- expansion:
  - happy path: new outcome -> new added child
	- otherwise: skip
- simulation: same as deterministic
- backpropagation: same as deterministic
*/

// TODO: test parallel mcts
/* spec:
- selection: 2 goroutines
  - same as sequential + loss applied
- expansion: 2 goroutines
  - same as sequential
- backpropagation: 2 goroutines
  - same as sequential + loss reversed
- selection + backup: 2 goroutines
	- either selection first or backup first then the other
*/
