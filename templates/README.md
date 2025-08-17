# Template Documentation

This directory contains template files for various AI operations in Gmail TUI. Templates use variable substitution to insert dynamic content.

## Template Structure

Templates support the following variable substitution patterns:
- Variables are enclosed in double curly braces: `{{variable_name}}`
- Variables are replaced with actual content during processing
- If a variable is not available, it will be replaced with an empty string

## Available Variables

### Core Email Variables
- `{{body}}` - The email content (plain text, stripped of HTML)
- `{{subject}}` - Email subject line
- `{{from}}` - Sender's email address
- `{{date}}` - Email date and time
- `{{to}}` - Primary recipient email address
- `{{cc}}` - CC recipients (comma-separated)
- `{{bcc}}` - BCC recipients (comma-separated)

### Technical Email Variables
- `{{reply-to}}` - Reply-to email address
- `{{message-id}}` - Gmail message ID
- `{{in-reply-to}}` - Reference to original message ID
- `{{references}}` - Email thread references
- `{{labels}}` - Gmail labels (comma-separated)

### AI-Specific Variables
- `{{max_words}}` - Maximum word limit for summaries (Slack templates)
- `{{wrap_width}}` - Text wrap width for formatting (Touch-up templates)

### Bulk Operation Variables
- `{{messages}}` - Combined content from multiple selected messages (recommended for bulk operations)
- `{{body}}` - Also works for bulk operations (legacy support)

### Obsidian Integration Variables
- `{{subject_slug}}` - URL-safe version of subject for filenames
- `{{from_domain}}` - Domain part of sender's email
- `{{ingest_date}}` - Date when email was ingested into Obsidian

## Template Types

### AI Templates (`/ai/`)

#### `summarize.md`
- **Purpose**: Generate email summaries
- **Key Variables**: `{{body}}`
- **Usage**: Triggered by `y` key or `:summarize` command

#### `reply.md`
- **Purpose**: Generate reply drafts
- **Key Variables**: `{{body}}`
- **Usage**: Triggered by `g` key or `:generate-reply` command

#### `label.md`
- **Purpose**: Suggest appropriate labels
- **Key Variables**: `{{body}}`, `{{labels}}`
- **Usage**: Triggered by `o` key or `:suggest-label` command

#### `touch_up.md`
- **Purpose**: Format content for terminal display
- **Key Variables**: `{{body}}`, `{{wrap_width}}`
- **Usage**: Automatic formatting during content display

### Slack Templates (`/slack/`)

#### `summary.md`
- **Purpose**: Generate summaries for Slack forwarding
- **Key Variables**: `{{body}}`, `{{max_words}}`, `{{subject}}`, `{{from}}`, `{{date}}`
- **Usage**: When forwarding emails to Slack with "summary" format

### Obsidian Templates (`/obsidian/`)

#### `email.md`
- **Purpose**: Format emails for Obsidian vault ingestion
- **Key Variables**: All email variables plus `{{ingest_date}}`
- **Usage**: When saving emails to Obsidian vault

## Configuration Priority

The template system follows this priority order:

1. **Template Files** (highest priority) - Specified in config as `*_template` file paths
2. **Inline Prompts** (medium priority) - Specified in config as `*_prompt` fields  
3. **Hardcoded Fallbacks** (lowest priority) - Built into the application

This file-first design ensures that template files always take precedence when specified, providing a cleaner and more predictable configuration experience.

## Best Practices

### Writing Templates
- Keep templates focused and specific to their purpose
- Use descriptive instructions that guide the AI model
- Include context about the expected output format
- Test templates with various email types

### Variable Usage
- Always include `{{body}}` for email content analysis
- Use `{{subject}}` and `{{from}}` for context when helpful
- For bulk operations, prefer `{{messages}}` over `{{body}}`
- Include `{{max_words}}` in summary templates for length control

### Template Maintenance
- Document any custom variables in comments
- Keep templates updated with new requirements
- Test template changes before deploying
- Maintain consistent style across related templates

## Examples

### Simple Summary Template
```markdown
Summarize this email in 3 bullet points:

{{body}}
```

### Contextual Reply Template
```markdown
Write a professional reply to this email from {{from}} with subject "{{subject}}":

{{body}}

Keep the same language as the original email.
```

### Label Suggestion Template
```markdown
Based on the content below, suggest up to 3 labels from this list: {{labels}}

Email content:
{{body}}

Return only a JSON array of label names.
```

## Troubleshooting

### Common Issues
- **Empty output**: Check if required variables are available
- **Missing content**: Verify variable names match exactly (case-sensitive)
- **Formatting problems**: Ensure proper line breaks and spacing in templates

### Variable Debugging
- Check the application logs for variable substitution details
- Use the configuration test mode to validate template loading
- Verify file paths are correct (relative to config directory)

For more information, see the main documentation in the repository README.