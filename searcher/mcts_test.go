package searcher

/* spec:
- selection:
	- happy path: fully expanded Node with stochastic moves (decision parent chance Node children) -> unvisited child or max UCB child
	- happy path: stochastic move Node (chance Node parent decision children), known outcome -> child Node
	- otherwise: skip
- expansion:
  - happy path: new outcome -> new added child
	- otherwise: skip
- simulation: same as deterministic
- backpropagation: same as deterministic
*/
