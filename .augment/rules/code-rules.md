---
type: "agent_requested"
description: "Code Rules"
---

# Rules

# Build

1. Utilize goreleaser and or taskfile to check and crosscheck, like build, test, etc.

# Golang Library Rules

1. use https://github.com/golang-standards/project-layout for project directory and naming standard

2. use https://github.com/spf13/cobra for the main cli library

3. use https://github.com/knadh/koanf for the config management

4. Use https://github.com/uber-go/zap for the structured logging

# Styles

1. config must can be parse yaml file, environment variables, and cli flags, with rules yaml lowest, cli flags highest

2. structured logging need to be output as json and reusable, and have easy switch between logging level, so when logging level is DEBUG it will print all, but when ERROR its only print error and in json

3.
