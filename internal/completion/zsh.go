package completion

import (
	"fmt"
	"strings"

	"github.com/gin31259461/homebase/internal/config"
	"github.com/gin31259461/homebase/internal/platform"
)

type candidate struct {
	key   string
	label string
}

func ZshForPlatform(platformID string, hooks []platform.SetupHook) (string, error) {
	groups, err := config.LoadPackageGroupsForPlatform(platformID)
	if err != nil {
		return "", err
	}
	tasks, err := config.LoadCleanupTasksForPlatform(platformID, nil)
	if err != nil {
		return "", err
	}

	groupCandidates := make([]candidate, 0, len(groups))
	for _, group := range groups {
		groupCandidates = append(groupCandidates, candidate{key: group.Key, label: group.Label})
	}
	taskCandidates := make([]candidate, 0, len(tasks))
	for _, task := range tasks {
		taskCandidates = append(taskCandidates, candidate{key: task.Key, label: task.Label})
	}
	hookCandidates := make([]candidate, 0, len(hooks))
	for _, hook := range hooks {
		hookCandidates = append(hookCandidates, candidate{key: hook.Key, label: hook.Label})
	}
	return zsh(groupCandidates, taskCandidates, hookCandidates), nil
}

func zsh(groups, tasks, hooks []candidate) string {
	var b strings.Builder
	b.WriteString(`#compdef hb

_hb() {
  local context state state_descr line command
  typeset -A opt_args
`)
	writeCandidates(&b, "commands", []candidate{
		{key: "bootstrap", label: "bootstrap dotfiles and platform basics"},
		{key: "install", label: "install package groups"},
		{key: "setup", label: "run or repair setup hooks"},
		{key: "cleanup", label: "clean caches and local clutter"},
		{key: "sync", label: "commit and push configured dotfiles"},
		{key: "config", label: "initialize Homebase configuration"},
		{key: "completion", label: "generate shell completion"},
		{key: "help", label: "show command usage"},
	})
	writeCandidates(&b, "groups", groups)
	writeCandidates(&b, "tasks", tasks)
	writeCandidates(&b, "hooks", hooks)
	b.WriteString(`
  if (( CURRENT == 2 )); then
    _describe 'command' commands
    return
  fi

  command=$words[2]
  words=($words[2,-1])
  (( CURRENT-- ))

  case $command in
    bootstrap)
      _arguments \
        '(-y --yes)'{-y,--yes}'[accept defaults and skip prompts]' \
        '--repo[dotfiles repository URL]:repository:' \
        '--install[run package installation after bootstrap]'
      ;;
    install)
      _arguments \
        '*--group[select a package group]:package group:->groups' \
        '--all[select all package groups]' \
        '(-y --yes)'{-y,--yes}'[skip confirmation]' \
        '--no-setup[skip post-install setup]' && return 0
      [[ $state == groups ]] && _describe 'package group' groups
      ;;
    setup)
      _arguments \
        '*--hook[select a setup hook]:setup hook:->hooks' \
        '--all[select all setup hooks]' \
        '(-y --yes)'{-y,--yes}'[skip confirmation]' && return 0
      [[ $state == hooks ]] && _describe 'setup hook' hooks
      ;;
    cleanup)
      _arguments \
        '*--task[select a cleanup task]:cleanup task:->tasks' \
        '--all[select all cleanup tasks]' \
        '(-y --yes)'{-y,--yes}'[skip confirmation]' && return 0
      [[ $state == tasks ]] && _describe 'cleanup task' tasks
      ;;
    sync)
      _arguments \
        '(-m --message)'{-m,--message}'[commit message]:message:' \
        '--no-push[commit without pushing]'
      ;;
    config)
      if (( CURRENT == 2 )); then
        _values 'config command' init
      elif [[ $words[2] == init ]]; then
        words=($words[2,-1])
        (( CURRENT-- ))
        _arguments '(-f --force)'{-f,--force}'[overwrite existing config]'
      fi
      ;;
    completion)
      (( CURRENT == 2 )) && _values 'shell' zsh
      ;;
  esac
}

_hb "$@"
`)
	return b.String()
}

func writeCandidates(b *strings.Builder, name string, values []candidate) {
	fmt.Fprintf(b, "  local -a %s=(\n", name)
	for _, value := range values {
		fmt.Fprintf(b, "    %s\n", zshQuote(value.key+":"+value.label))
	}
	b.WriteString("  )\n")
}

func zshQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
