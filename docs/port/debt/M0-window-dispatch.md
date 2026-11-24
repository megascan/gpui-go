# M0-window-dispatch debt

- window_dispatch.go `spawnWindowTask`: works around `Window.Spawn` dropping the task Scope (see handoff); replace with `Window.Spawn` when fixed.
- window_smoke_test.go releases the window root by hand because window removal does not release it (see handoff).
- window_prompts.go `WithView`: an unanswered prompt whose view is released never closes the oneshot sender, so the receiver is not cancelled as upstream's drop does.
