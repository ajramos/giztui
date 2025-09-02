# Gmail Search Reference

## Overview

Gmail provides powerful search operators that can be used both for searching messages in GizTUI and creating Gmail filters. This reference covers the most useful operators with practical examples.

## Basic Search Operators

| Operator           | Description                                     | Example                       |
|--------------------|-------------------------------------------------|-------------------------------|
| `from:`            | Filter by sender                                | `from:ana@gmail.com`          |
| `to:`              | Filter by recipient                             | `to:info@empresa.com`         |
| `cc:` / `bcc:`     | Filter by carbon copy / blind carbon copy       | `cc:jefe@empresa.com`         |
| `subject:`         | Words in the subject                            | `subject:invoice`             |
| `has:attachment`   | Only emails with attachments                    | `has:attachment`              |
| `filename:`        | Attachment name or type                         | `filename:pdf`                |
| `in:`              | Special folder or specific label                | `in:archive` `in:spam`        |
| `label:`           | Messages with a specific label                  | `label:work`                  |
| `category:`        | Gmail categories (Primary, Social, etc.)        | `category:social`             |
| `is:`              | Special state: read, unread, important, etc.    | `is:unread` `is:important`    |

## Date and Time Operators

| Operator           | Description                                     | Example                       |
|--------------------|-------------------------------------------------|-------------------------------|
| `before:`          | Emails before a date (YYYY/MM/DD)               | `before:2025/01/01`           |
| `after:`           | Emails from a date                              | `after:2025/05/01`            |
| `older_than:`      | Older than (days, months, years)                | `older_than:1y`               |
| `newer_than:`      | Newer than (days, months, years)                | `newer_than:7d`               |

## Content and Attachment Operators

| Operator           | Description                                     | Example                       |
|--------------------|-------------------------------------------------|-------------------------------|
| `has:document`     | Google Docs-type attachments                    | `has:document`                |
| `has:youtube`      | Emails containing YouTube links                 | `has:youtube`                 |
| `has:calendar`     | Emails with calendar invitations                | `has:calendar`                |
| `size:`            | Emails of exact size                            | `size:5000000`                |
| `larger:`          | Emails larger than specified size               | `larger:10M`                  |
| `smaller:`         | Emails smaller than specified size              | `smaller:5M`                  |
| `list:`            | Search by mailing list                          | `list:newsletter@company.com` |

## Logical Operators

- **AND** — Combine two criteria (must be uppercase)
  - Example: `from:ana@gmail.com AND has:attachment`
- **OR** — Either one criterion or the other
  - Example: `from:ana OR from:pepe`
- **-** — Exclude words or phrases
  - Example: `-subject:ads`
- **"phrase"** — Exact phrase search
  - Example: `"march invoice"`
- **()** — Group terms
  - Example: `(invoice receipt)`
- **\*** — Wildcard for any word or domain
  - Example: `*@empresa.com`

## Advanced Search Examples

### Complex Queries

```
from:support@company.com subject:(invoice payment) has:attachment after:2025/01/01
```

```
(from:client1@company.com OR from:client2@company.com) AND has:attachment -subject:spam
```

```
label:work is:unread newer_than:3d
```

### Size-based Searches

```
larger:25M has:attachment
```

```
smaller:1M is:read older_than:1y
```

### Content-specific Searches

```
has:youtube category:social
```

```
filename:pdf subject:report after:2024/12/01
```

## Special Search Locations

- `in:anywhere` — Search everywhere, including Spam/Trash
- `in:sent` — Search sent messages
- `in:drafts` — Search draft messages
- `in:inbox` — Search inbox messages
- `in:trash` — Search trash
- `in:spam` — Search spam folder

## Using Search in GizTUI

1. Press `/` to open search
2. Enter your search query using the operators above
3. Press Enter to execute the search
4. Use arrow keys to navigate results
5. Press `Esc` to return to inbox

## Creating Gmail Filters

To create automatic filters based on these operators:

1. Use Gmail's advanced search (funnel icon)
2. Fill in the search criteria
3. Click "Create filter"
4. Choose actions: mark as read, move to folder, apply label, delete, forward, etc.

## Tips and Best Practices

- **Save frequent searches**: Create saved search queries in GizTUI for commonly used searches
- **Combine operators**: Use multiple operators for precise filtering
- **Test first**: Test search queries before creating automated filters
- **Use wildcards**: Leverage `*` for domain-wide searches
- **Date formats**: Always use YYYY/MM/DD format for date searches
- **Size units**: Use standard units (K, M, G) for size operators

---

These search operators work consistently across Gmail's web interface, mobile apps, and GizTUI, making them a powerful tool for email management and organization.