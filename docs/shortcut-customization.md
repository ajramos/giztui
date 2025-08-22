# Shortcut Customization Guide

Gmail TUI now supports fully customizable keyboard shortcuts through the configuration file. You can personalize your experience by mapping actions to keys that feel natural to you.

## Overview

The shortcut customization system allows you to:
- Override any default keyboard shortcut
- Create custom key mappings for your workflow
- Maintain backward compatibility with existing shortcuts
- Use different shortcuts for different environments

## Configuration

All shortcuts are configured in the `keys` section of your `config.json` file:

```json
{
  "keys": {
    "summarize": "y",
    "generate_reply": "g",
    "suggest_label": "o",
    "reply": "r",
    "compose": "n",
    "refresh": "R",
    "search": "s",
    "unread": "u",
    "toggle_read": "t",
    "trash": "d",
    "archive": "a",
    "move": "m",
    "prompt": "p",
    "drafts": "D",
    "attachments": "A",
    "manage_labels": "l",
    "quit": "q",
    
    "obsidian": "O",
    "slack": "K",
    "markdown": "M",
    "save_message": "w",
    "save_raw": "W",
    "rsvp": "V",
    "link_picker": "L",
    "bulk_mode": "v",
    "command_mode": ":",
    "help": "?"
  }
}
```

## Available Actions

### Core Email Operations
- `summarize` - Generate AI summary of selected message
- `generate_reply` - Generate AI-powered reply
- `suggest_label` - Get AI label suggestions
- `reply` - Reply to selected message
- `compose` - Compose new message
- `load_more` - Load next 50 messages
- `refresh` - Refresh message list
- `search` - Open search overlay
- `unread` - Show unread messages
- `toggle_read` - Mark message as read/unread
- `trash` - Move message to trash
- `archive` - Archive message
- `move` - Move message to folder/label
- `prompt` - Apply AI prompt to message
- `drafts` - Show draft messages
- `attachments` - Show message attachments
- `manage_labels` - Manage message labels
- `quit` - Exit application

### Additional Features
- `obsidian` - Send message to Obsidian
- `slack` - Forward message to Slack
- `markdown` - Toggle markdown rendering
- `save_message` - Save message to file
- `save_raw` - Save raw EML file
- `rsvp` - Toggle RSVP panel
- `link_picker` - Open link picker
- `bulk_mode` - Toggle bulk selection mode
- `command_mode` - Open command bar
- `help` - Toggle help display

## Customization Examples

### Vim-Style Shortcuts
```json
{
  "keys": {
    "compose": "i",
    "search": "/",
    "quit": ":q",
    "help": ":h"
  }
}
```

### Emacs-Style Shortcuts
```json
{
  "keys": {
    "compose": "C-x",
    "search": "C-s",
    "quit": "C-x C-c"
  }
}
```

### Numeric Shortcuts
```json
{
  "keys": {
    "summarize": "1",
    "generate_reply": "2",
    "suggest_label": "3",
    "reply": "4"
  }
}
```

### Function Key Shortcuts
```json
{
  "keys": {
    "compose": "F1",
    "search": "F2",
    "help": "F12"
  }
}
```

## How It Works

1. **Priority System**: Configurable shortcuts take precedence over hardcoded defaults
2. **Fallback**: If a key isn't configured, it falls back to the default behavior
3. **Conflict Resolution**: No two actions can use the same key
4. **Hot Reload**: Changes take effect after restarting the application

## Best Practices

### Choose Intuitive Keys
- Use keys that relate to the action (e.g., 'r' for reply, 'n' for new)
- Consider your existing muscle memory from other applications
- Avoid conflicts with navigation keys (arrow keys, Tab, Enter, Esc)

### Maintain Consistency
- Use similar patterns across related actions
- Keep frequently used shortcuts easily accessible
- Consider ergonomics for your keyboard layout

### Document Your Setup
- Keep a backup of your custom configuration
- Share useful configurations with your team
- Version control your config files

## Troubleshooting

### Shortcut Not Working
1. Check that the key is properly configured in `config.json`
2. Ensure the key doesn't conflict with system shortcuts
3. Verify the configuration file syntax is valid
4. Restart the application after making changes

### Conflicts with Default Behavior
- Configurable shortcuts always take precedence
- If you want to restore default behavior, remove the custom mapping
- Check the logs for any error messages

### Performance Considerations
- Single-character shortcuts are processed most efficiently
- Complex key combinations may have slight delays
- Avoid mapping too many shortcuts to the same key

## Migration from Default Shortcuts

If you're used to the default shortcuts, you can:
1. Start with a few customizations
2. Gradually adapt your workflow
3. Keep a reference of your changes
4. Test new shortcuts before committing to them

## Advanced Configuration

### Environment-Specific Shortcuts
You can maintain different configuration files for different environments:

```bash
# Development environment
cp config.json config-dev.json

# Production environment  
cp config.json config-prod.json

# Custom shortcuts for specific workflows
cp config.json config-custom.json
```

### Shortcut Aliases
Some actions can be triggered by multiple shortcuts by duplicating the configuration:

```json
{
  "keys": {
    "compose": "n",
    "new_message": "n"  // Same action, different name
  }
}
```

## Contributing

If you have ideas for new configurable shortcuts:
1. Check if the action already exists
2. Propose the new shortcut in an issue
3. Consider backward compatibility
4. Document the new functionality

## Support

For help with shortcut customization:
1. Check this documentation
2. Review the example configuration files
3. Search existing issues
4. Create a new issue with your specific problem

---

**Note**: Shortcut customization is a powerful feature that can significantly improve your productivity. Take time to experiment and find what works best for your workflow!