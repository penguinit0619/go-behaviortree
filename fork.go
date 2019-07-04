package behaviortree

import (
	"fmt"
	"sync"
)

// Fork generates a stateful Tick which will tick all children at once, returning after all children return a result,
// returning running if any children did so, and ticking only those which returned running in subsequent calls, until
// all children have returned a non-running status, combining any errors, and returning success if there were no
// failures or errors (otherwise failure), repeating this cycle for subsequent ticks
func Fork() Tick {
	var (
		mutex     sync.Mutex
		remaining []Node
		status    Status
		err       error
	)
	return func(children []Node) (Status, error) {
		mutex.Lock()
		defer mutex.Unlock()

		if status == 0 && err == nil {
			// cycle start
			status = Success
			remaining = make([]Node, len(children))
			copy(remaining, children)
		}

		count := len(remaining)
		outputs := make(chan func(), count)
		for _, node := range remaining {
			go func(node Node) {
				rs, re := node.Tick()
				outputs <- func() {
					if re != nil {
						rs = Failure
						if err != nil {
							err = fmt.Errorf("%s | %s", err.Error(), re.Error())
						} else {
							err = re
						}
					}
					switch rs {
					case Running:
						remaining = append(remaining, node)
					case Success:
						// success is the initial status (until 1+ failures)
					default:
						status = Failure
					}
				}
			}(node)
		}
		remaining = remaining[:0]
		for x := 0; x < count; x++ {
			(<-outputs)()
		}

		if len(remaining) == 0 {
			// cycle end
			rs, re := status, err
			status, err = 0, nil
			return rs, re
		}

		return Running, nil
	}
}
