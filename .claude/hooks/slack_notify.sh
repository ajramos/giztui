#!/usr/bin/env bash
set -euo pipefail

# Minimal Slack notifier for Claude Code hooks (Notification, Stop)
# Uses SLACK_WEBHOOK_URL from environment or from this project's .claude/hooks/slack.env

EVENT=${CLAUDE_HOOK_EVENT:-${1:-Notification}}
PROJECT_DIR=${CLAUDE_PROJECT_DIR:-$(pwd)}

# Try env first; fallback to project-local config file
WEBHOOK_URL=${SLACK_WEBHOOK_URL:-}
if [[ -z "${WEBHOOK_URL}" ]]; then
  LOCAL_CFG="${PROJECT_DIR}/.claude/hooks/slack.env"
  if [[ -f "${LOCAL_CFG}" ]]; then
    # shellcheck disable=SC1090
    source "${LOCAL_CFG}"
    WEBHOOK_URL=${SLACK_WEBHOOK_URL:-}
  fi
fi

if [[ -z "${WEBHOOK_URL}" ]]; then
  echo "SLACK_WEBHOOK_URL not configured; skipping notification" >&2
  exit 0
fi

# Read hook payload from stdin (may be empty)
RAW_INPUT="$(cat || true)"

# Debug: log the raw input to see what we're getting
echo "DEBUG: Hook received payload: ${RAW_INPUT}" >> /tmp/claude_hook_debug.log
echo "DEBUG: Event: ${EVENT}" >> /tmp/claude_hook_debug.log
echo "DEBUG: Timestamp: $(date)" >> /tmp/claude_hook_debug.log
echo "---" >> /tmp/claude_hook_debug.log

# Enhanced information extraction from hook payload
summary=""
tool_name=""
command=""
working_dir=""
session_id=""
if command -v jq >/dev/null 2>&1; then
  summary=$(echo "${RAW_INPUT}" | jq -r '(.message // .status // .final_text // .description // empty) | strings' 2>/dev/null || true)
  tool_name=$(echo "${RAW_INPUT}" | jq -r '(.tool_name // .tool // empty) | strings' 2>/dev/null || true)
  command=$(echo "${RAW_INPUT}" | jq -r '(.tool_input.command // .tool_input.pattern // .tool_input.file_path // .command // empty) | strings' 2>/dev/null || true)
  working_dir=$(echo "${RAW_INPUT}" | jq -r '(.tool_input.path // .cwd // empty) | strings' 2>/dev/null || true)
  session_id=$(echo "${RAW_INPUT}" | jq -r '(.session_id // empty) | strings' 2>/dev/null || true)

  # Also try to extract from transcript if available
  transcript_path=$(echo "${RAW_INPUT}" | jq -r '(.transcript_path // empty) | strings' 2>/dev/null || true)
  if [[ -n "${transcript_path}" && -f "${transcript_path}" ]]; then
    # Get the last few lines from transcript to extract tool info
    # Look for assistant messages with tool_use content
    last_tool=$(tail -10 "${transcript_path}" | jq -r 'select(.type == "assistant") | .message.content[]? | select(.type == "tool_use") | .name' 2>/dev/null | tail -1 || true)
    last_command=$(tail -10 "${transcript_path}" | jq -r 'select(.type == "assistant") | .message.content[]? | select(.type == "tool_use") | .input.command' 2>/dev/null | tail -1 || true)
    last_description=$(tail -10 "${transcript_path}" | jq -r 'select(.type == "assistant") | .message.content[]? | select(.type == "tool_use") | .input.description' 2>/dev/null | tail -1 || true)

    if [[ -n "${last_tool}" && -z "${tool_name}" ]]; then
      tool_name="${last_tool}"
    fi
    if [[ -n "${last_command}" && -z "${command}" ]]; then
      command="${last_command}"
    fi
    if [[ -n "${last_description}" && -z "${summary}" ]]; then
      summary="${last_description}"
    fi
  fi
fi

timestamp=$(date '+%Y-%m-%d %H:%M:%S')
project_name=$(basename "${PROJECT_DIR}")
branch=$(git -C "${PROJECT_DIR}" branch --show-current 2>/dev/null || echo "unknown")

# Build detailed notification with proper newlines
title="🤖 Claude Code: ${EVENT}"
text="${title}"$'\n'"📁 Project: ${project_name}"$'\n'"🌿 Branch: ${branch}"$'\n'"⏰ ${timestamp}"

if [[ -n "${session_id}" ]]; then
  short_session="${session_id:0:8}..."
  text+=$'\n'"🔗 Session: ${short_session}"
fi

if [[ -n "${tool_name}" ]]; then
  text+=$'\n'"🔧 Tool: ${tool_name}"
fi

if [[ -n "${command}" ]]; then
  # Truncate long commands
  if [[ ${#command} -gt 80 ]]; then
    command="${command:0:77}..."
  fi
  text+=$'\n'"💻 Command: \`${command}\`"
fi

if [[ -n "${working_dir}" && "${working_dir}" != "${PROJECT_DIR}" ]]; then
  text+=$'\n'"📂 Path: ${working_dir}"
fi

if [[ -n "${summary}" ]]; then
  # Truncate long summaries
  if [[ ${#summary} -gt 200 ]]; then
    summary="${summary:0:197}..."
  fi
  text+=$'\n'"📝 Details: ${summary}"
fi

# Build JSON safely via jq if available
if command -v jq >/dev/null 2>&1; then
  payload=$(jq -Rn --arg t "$text" '{text: $t}')
else
  # Fallback: very simple JSON (may break on special chars)
  payload="{\"text\": \"${text//\"/\\\"}\"}"
fi

curl -sS -X POST -H 'Content-type: application/json' --data "${payload}" "${WEBHOOK_URL}" >/dev/null || true

exit 0
