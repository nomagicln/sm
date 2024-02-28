# sm - StateMachine

[![Go Reference](https://pkg.go.dev/badge/github.com/nomagicln/sm.svg)](https://pkg.go.dev/github.com/nomagicln/sm)
[![codecov](https://codecov.io/gh/nomagicln/sm/graph/badge.svg?token=IYM19YGS7C)](https://codecov.io/gh/nomagicln/sm)

```go
package main

import (
	"github.com/nomagicln/sm"
)

func main() {
	machine := sm.NewStateMachine(
		"PackageManager",
		[]sm.Transition{
			sm.NewTransition("Unchecked", "Ready"),    // Go to Ready after checking
			sm.NewTransition("Ready", "Installing"),   // Try install, then go to Installing
			sm.NewTransition("Ready", "Damaged"),      // Try install, but failed, then go to Damaged
			sm.NewTransition("Installing", "Running"), // Install success, then go to Running
			sm.NewTransition("Installing", "Failed"),  // Install failed, then go to Failed
			sm.NewTransition("Running", "Ready"),      // Uninstall, then go to Ready
			sm.NewTransition("Failed", "Damaged"),     // Check what's wrong, if can't fix, then go to Damaged
			sm.NewTransition("Failed", "Ready"),       // Check what's wrong, if can fix, then go to Ready
		},
		"Unchecked",
	) // Initial state

	if err := machine.Transition("Ready", nil); err != nil {
		panic(err)
	}

	if err := machine.Transition("Ready", nil); err != nil {
		panic(err)
	}
}
```

Import:

```bash
go get github.com/nomagicln/sm
```
