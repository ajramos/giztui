## Gmail Filters and Search Operators: Crash Course

Gmail filters are built on advanced search operators that you can use both to automatically filter incoming emails and to quickly search for specific messages from the search bar.

Below is a practical summary of the most useful ones, with real examples.

### Basic operators for filters

| Operator           | What it does                                   | Example                       |
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
| `before:`          | Emails before a date (YYYY/MM/DD)               | `before:2025/01/01`           |
| `after:`           | Emails from a date                              | `after:2025/05/01`            |
| `older_than:`      | Older than (days, months, years)                | `older_than:1y`               |
| `newer_than:`      | Newer than (days, months, years)                | `newer_than:7d`               |
| `has:document`     | Google Docs-type attachments                    | `has:document`                |
| `has:youtube`      | Emails containing YouTube links                 | `has:youtube`                 |

### Logical and advanced operators

- **AND** — Combine two criteria (must be uppercase).
  - Example: `from:ana@gmail.com AND has:attachment`
- **OR** — Either one criterion or the other.
  - Example: `from:ana OR from:pepe`
- **-** — Exclude words or phrases.
  - Example: `-subject:ads`
- **"word or phrase"** — Exact phrase search.
  - Example: `"march invoice"`
- **Parentheses ()** — Group terms.
  - Example: `(invoice receipt)`
- **Asterisk \*** — Wildcard for any word or domain.
  - Example: `*@empresa.com`

### Useful special filters

- `has:calendar` — Emails with calendar invitations.
- `size:5000000` — Emails exactly 5 MB in size.
- `larger:10M` / `smaller:5M` — Emails larger/smaller than a size.
- `list:boletin@empresa.com` — Search by mailing list.
- `in:anywhere` — Search anywhere, including Spam/Trash.

### How to use them

- You can combine them for ultra-precise searches:
  - `from:soporte@empresa.com subject:(invoice payment) has:attachment after:2025/01/01`
- To create an automatic filter, use the advanced search (funnel icon), fill in the fields, click "Create filter", and choose actions: mark as read, move to folder, apply label, delete, forward, etc.

### Tip

Save frequent searches or filters as custom shortcuts from the Gmail interface to automate email organization.


