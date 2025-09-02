# 🚀 Getting Started with GizTUI

Welcome to GizTUI! This guide will help you get up and running quickly with your new AI-powered terminal Gmail client.

## 📋 What You'll Need

Before starting, make sure you have:

1. **Gmail Account** - A Google/Gmail account you want to manage
2. **Terminal with 256-color support** - Most modern terminals work fine
3. **Gmail API Credentials** - We'll help you set this up below
4. **Ollama** (optional) - For local AI features ([Install Ollama](https://ollama.ai/))

## 📦 Installation

### Option 1: Download Pre-built Binary (Recommended)

Choose your platform and download from [GitHub Releases](https://github.com/ajramos/giztui/releases/latest):

**Linux/macOS:**
```bash
# Download and extract (replace with your platform)
curl -L https://github.com/ajramos/giztui/releases/latest/download/giztui-linux-amd64.tar.gz | tar -xz

# Move to PATH
sudo mv giztui /usr/local/bin/

# Verify installation
giztui --version
```

**Windows:**
1. Download `giztui-windows-amd64.zip`
2. Extract to a folder (e.g., `C:\tools\giztui\`)
3. Add the folder to your PATH environment variable
4. Open Command Prompt or PowerShell and verify: `giztui --version`

### Option 2: Install with Go

```bash
go install github.com/ajramos/giztui/cmd/giztui@latest
```

### Option 3: Build from Source

```bash
git clone https://github.com/ajramos/giztui.git
cd giztui
make build
./build/giztui --version
```

## 🔑 Gmail API Setup

GizTUI needs Gmail API credentials to access your email. Don't worry - your data stays private and local!

### Step 1: Enable Gmail API

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project or select an existing one
3. Enable the Gmail API:
   - Go to **APIs & Services** → **Library**
   - Search for "Gmail API" and click **Enable**

### Step 2: Create Credentials

1. Go to **APIs & Services** → **Credentials**
2. Click **Create Credentials** → **OAuth client ID**
3. If prompted, configure the OAuth consent screen:
   - Choose **External** user type
   - Fill in basic app information (just for you to see)
   - Add your email to test users
4. For Application type, choose **Desktop application**
5. Name it something like "GizTUI Gmail Client"
6. Click **Create**

### Step 3: Download Credentials

1. Click the download button (⬇️) next to your new OAuth client
2. Save the file as `credentials.json`
3. Place it in your GizTUI config directory:
   - **Linux/macOS**: `~/.config/giztui/credentials.json`
   - **Windows**: `%APPDATA%\giztui\credentials.json`

## ⚡ First Run

Now you're ready to start GizTUI!

### 1. Run Setup

```bash
giztui --setup
```

This interactive setup will:
- Detect your credentials file
- Walk you through OAuth authentication
- Create a basic configuration file
- Test your connection

### 2. First Launch

```bash
giztui
```

You should see:
1. **Welcome screen** with your account info
2. **Loading inbox** indicator
3. Your **Gmail inbox** appearing shortly

### 3. Take a Tour

Try these essential operations:

**Navigation:**
- `↑↓` - Navigate messages
- `Enter` - Read selected message
- `q` - Quit application
- `?` - Show help/shortcuts

**Basic Email Operations:**
- `r` - Mark as read/unread
- `a` - Archive message
- `d` - Move to trash
- `u` - View unread messages

**Search and Commands:**
- `s` - Search emails
- `:` - Enter command mode
- `/` - Search within current message

## 🎯 Your First 5 Minutes

Here's a quick tour to get familiar:

### 1. Browse Your Inbox (1 minute)
- Use arrow keys to navigate through messages
- Press `Enter` to open a message
- Press `Esc` to go back to the message list

### 2. Try Basic Operations (2 minutes)
- Select a message and press `a` to archive it
- Press `u` to see unread messages
- Try marking a message as read with `r`

### 3. Search Your Email (1 minute)
- Press `s` to search
- Try searching for "unread" or a sender's name
- Press `Esc` to return to normal view

### 4. Explore Commands (1 minute)
- Press `:` to enter command mode
- Type `help` and press Enter
- Try `:labels` to see all your Gmail labels
- Press `Esc` to exit command mode

### 5. Get Help (30 seconds)
- Press `?` to see the help screen with keyboard shortcuts
- Press `Esc` to close help

## 🧠 Enable AI Features (Optional)

To use GizTUI's AI capabilities for email summaries and smart features:

### Option 1: Local AI with Ollama (Recommended)

1. **Install Ollama**: Follow instructions at [ollama.ai](https://ollama.ai/)

2. **Download a model**:
   ```bash
   ollama pull llama2  # or llama3, mistral, etc.
   ```

3. **Configure GizTUI**: Edit `~/.config/giztui/config.json`:
   ```json
   {
     "llm": {
       "provider": "ollama",
       "ollama": {
         "model": "llama2",
         "base_url": "http://localhost:11434"
       }
     }
   }
   ```

4. **Test AI features**:
   - Select a message and press `Shift+S` for AI summary
   - Try the prompt library with `Shift+P`

### Option 2: Cloud AI with Amazon Bedrock

If you have AWS credentials configured:

1. **Configure Bedrock**: Edit `config.json`:
   ```json
   {
     "llm": {
       "provider": "bedrock",
       "bedrock": {
         "model": "anthropic.claude-3-sonnet-20240229-v1:0",
         "region": "us-east-1"
       }
     }
   }
   ```

2. **Ensure AWS credentials** are configured (via AWS CLI, environment variables, or IAM role)

## 🔧 Essential Configuration

Create or edit `~/.config/giztui/config.json` for basic customization:

```json
{
  "gmail": {
    "max_results": 50,
    "timeout": "30s"
  },
  "ui": {
    "theme": "slate-blue",
    "layout": {
      "default_breakpoint": "wide"
    }
  },
  "shortcuts": {
    "help": "?",
    "quit": "q",
    "search": "s",
    "unread": "u"
  },
  "llm": {
    "provider": "ollama",
    "timeout": "2m"
  }
}
```

## 🆘 Common Issues & Solutions

### "Credentials not found"
- Make sure `credentials.json` is in `~/.config/giztui/`
- Check file permissions (should be readable)
- Verify the JSON format is valid

### "Access blocked: This app isn't verified"
- This is normal for personal use
- Click "Advanced" → "Go to [your app name] (unsafe)"
- This is safe when using your own OAuth credentials

### "No messages loading"
- Check your internet connection
- Verify Gmail API is enabled in Google Cloud Console
- Try running `giztui --setup` again

### AI features not working
- Ensure Ollama is running: `ollama list`
- Check the model name in your config matches exactly
- For Bedrock, verify your AWS credentials: `aws sts get-caller-identity`

## 📚 What's Next?

Now that you're up and running:

1. **Explore Features**: Check out [FEATURES.md](FEATURES.md) for the complete feature list
2. **Customize**: Read [CONFIGURATION.md](CONFIGURATION.md) for detailed customization
3. **Master Shortcuts**: Review [KEYBOARD_SHORTCUTS.md](KEYBOARD_SHORTCUTS.md) for efficiency
4. **Advanced Usage**: Try integrations like Slack forwarding and Obsidian note-taking

## 🔗 Quick Reference

**Essential Shortcuts:**
- `?` - Help/shortcuts
- `:` - Command mode  
- `q` - Quit
- `s` - Search
- `u` - Unread messages
- `a` - Archive
- `r` - Toggle read/unread
- `Shift+S` - AI summary
- `O` - Open in Gmail web

**Essential Commands:**
- `:help` - Show help
- `:search <query>` - Search emails
- `:labels` - Show all labels
- `:themes` - List available themes
- `:quit` - Exit application

---

Welcome to GizTUI! You're now ready to manage your Gmail efficiently from the terminal. 

🎉 **Happy emailing!**