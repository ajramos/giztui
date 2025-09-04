# Long Recipient List Fix - Smart Field Truncation

## Problem Statement

When emails have very long "To" or "Cc" recipient lists, the header fields wrap to many lines but are capped at a maximum of 12 lines in `adjustHeaderHeight()`. This causes important fields like Labels, Attachments, and Date to be cut off and not visible to the user.

**Root Cause:** Header height limitation in `internal/tui/messages.go:2843`
```go
if lines > maxHeight {
    lines = maxHeight  // Capped at 12 lines!
}
```

## Solution: Smart Field Truncation (Option 2)

Implement intelligent truncation of "To" and "Cc" fields to ensure other important header fields remain visible.

## Implementation Plan

### Phase 1: Core Truncation Logic
**File: `internal/render/email.go`**

#### 1.1 Add Truncation Function
```go
// truncateRecipientField truncates recipient fields to fit within specified line limit
func (er *EmailRenderer) truncateRecipientField(fieldName, value string, maxLines int, lineWidth int) string {
    if maxLines <= 0 {
        maxLines = 3 // Default to 3 lines for recipient fields
    }
    
    // Calculate available width for content (excluding field name)
    prefix := fieldName + ": "
    availableWidth := lineWidth - len(prefix)
    if availableWidth < 20 {
        availableWidth = 20 // Minimum reasonable width
    }
    
    // Split recipients by comma and trim whitespace
    recipients := make([]string, 0)
    for _, recipient := range strings.Split(value, ",") {
        if trimmed := strings.TrimSpace(recipient); trimmed != "" {
            recipients = append(recipients, trimmed)
        }
    }
    
    if len(recipients) == 0 {
        return ""
    }
    
    // Calculate how many recipients fit within maxLines
    currentLine := 1
    currentLineLength := 0
    fittingRecipients := 0
    
    for i, recipient := range recipients {
        recipientLength := len(recipient)
        if i > 0 {
            recipientLength += 2 // Add ", " separator
        }
        
        // Check if this recipient fits on current line
        if currentLineLength + recipientLength <= availableWidth {
            currentLineLength += recipientLength
            fittingRecipients++
        } else {
            // Need new line
            if currentLine >= maxLines {
                break // Hit line limit
            }
            currentLine++
            currentLineLength = recipientLength
            fittingRecipients++
        }
    }
    
    // Build result string
    if fittingRecipients >= len(recipients) {
        // All recipients fit
        return strings.Join(recipients, ", ")
    }
    
    // Truncation needed
    truncatedRecipients := recipients[:fittingRecipients]
    remaining := len(recipients) - fittingRecipients
    
    result := strings.Join(truncatedRecipients, ", ")
    if remaining > 0 {
        suffix := fmt.Sprintf(" ... and %d more recipient", remaining)
        if remaining > 1 {
            suffix += "s"
        }
        
        // Ensure suffix fits on last line or new line
        if len(result) + len(suffix) <= availableWidth {
            result += suffix
        } else {
            // Put suffix on new line if we haven't hit maxLines
            if currentLine < maxLines {
                result += "\n" + strings.Repeat(" ", len(prefix)) + suffix
            } else {
                // Replace last recipient with suffix
                if len(truncatedRecipients) > 0 {
                    lastRecipient := truncatedRecipients[len(truncatedRecipients)-1]
                    if len(result) - len(lastRecipient) + len(suffix) <= availableWidth {
                        result = strings.Join(truncatedRecipients[:len(truncatedRecipients)-1], ", ") + suffix
                        remaining++
                    }
                }
            }
        }
    }
    
    return result
}
```

#### 1.2 Modify writeWrappedHeaderField Function
```go
// Update writeWrappedHeaderField to use truncation for recipient fields
func (er *EmailRenderer) writeWrappedHeaderField(b *strings.Builder, fieldName, value string, width int) {
    if strings.TrimSpace(value) == "" {
        return
    }

    // Special handling for recipient fields - truncate to max 3 lines
    if fieldName == "To" || fieldName == "Cc" {
        truncatedValue := er.truncateRecipientField(fieldName, value, 3, width)
        if truncatedValue == "" {
            return
        }
        value = truncatedValue
    }

    prefix := fieldName + ": "
    prefixLen := len(prefix)

    // If the entire line fits, write it as-is
    if prefixLen+len(value) <= width {
        fmt.Fprintf(b, "%s%s\n", prefix, value)
        return
    }

    // Existing line wrapping logic continues...
    // [Keep existing implementation for non-recipient fields]
}
```

### Phase 2: Configuration Options
**File: `internal/config/config.go`**

Add configurable truncation settings:
```go
type DisplayConfig struct {
    // ... existing fields ...
    MaxRecipientLines int `yaml:"max_recipient_lines" json:"max_recipient_lines"`
}

// Default values
func DefaultDisplayConfig() DisplayConfig {
    return DisplayConfig{
        // ... existing defaults ...
        MaxRecipientLines: 3,
    }
}
```

### Phase 3: Testing Integration Points
**Files to verify integration:**
- `internal/tui/markdown.go:114` - Uses `FormatHeaderPlainWithWidth`
- `internal/tui/messages.go:2281` - Uses `FormatHeaderPlain` (save function)
- All email display code paths that render headers

## Comprehensive Test Plan

### Test Case 1: Short Recipient Lists (Baseline)
**Input:**
- To: "user@example.com"
- To: "user1@example.com, user2@example.com"

**Expected:**
- No truncation applied
- All recipients displayed fully
- Other header fields (Subject, From, Date, Labels) visible

**Verification:**
- Header height ≤ 6 lines
- All fields visible in UI
- No "... and X more" indicators

### Test Case 2: Medium Recipient Lists
**Input:**
- To: 8-12 recipients (names of varying lengths)
- Cc: 3-5 recipients

**Expected:**
- To field truncated to 3 lines with "... and X more recipients"
- Cc field fits completely (under 3 lines)
- Labels, Date, and other fields remain visible

**Verification:**
- Header height ≤ 12 lines
- Recipient count math is correct ("... and 4 more recipients")
- Other fields not cut off

### Test Case 3: Very Long Recipient Lists
**Input:**
- To: 50+ recipients with long email addresses
- Cc: 20+ recipients
- Subject: Long subject line
- Labels: Multiple labels

**Expected:**
- To field: exactly 3 lines + "... and X more recipients"
- Cc field: exactly 3 lines + "... and X more recipients" 
- Subject: wraps normally (not truncated)
- Labels: fully visible
- Date: fully visible

**Verification:**
- Header height ≤ 15 lines total
- Math: displayed recipients + "X more" = total count
- Critical fields (Labels, Date) always visible

### Test Case 4: Edge Cases

#### 4.1 Single Very Long Email Address
**Input:**
- To: "very-very-very-long-email-address-that-exceeds-line-width@very-long-domain-name.example.com"

**Expected:**
- Single recipient displayed with line wrapping
- No truncation indicator (only 1 recipient)

#### 4.2 Recipients with Special Characters
**Input:**
- To: "User Name <user@example.com>, "John Doe" <john@example.com>"

**Expected:**
- Proper parsing despite angle brackets and quotes
- Correct recipient counting

#### 4.3 Mixed Short and Long Recipients
**Input:**
- To: "a@b.co, very-long-email-address@very-long-domain.com, c@d.co, another-very-long-address@domain.example.com"

**Expected:**
- Intelligent line fitting
- Correct truncation at line boundaries

### Test Case 5: Configuration Changes
**Input:**
- Change `max_recipient_lines` from 3 to 2
- Change from 3 to 5
- Set to 0 (should use default of 3)

**Expected:**
- Truncation adapts to new line limits
- No crashes or UI issues
- Consistent behavior across restarts

### Test Case 6: Different Terminal Widths
**Test Environments:**
- 80 column terminal
- 120 column terminal  
- 200 column terminal
- Very narrow (60 column) terminal

**Expected:**
- Truncation adapts to available width
- Recipient counting remains accurate
- UI remains usable at all widths

### Test Case 7: Integration Testing
**Scenarios:**
1. **LLM Mode ON**: Header formatting with LLM touch-up enabled
2. **LLM Mode OFF**: Standard header formatting
3. **Header Toggle**: Hide/show headers (press 'h')
4. **Theme Changes**: Header colors and formatting
5. **Save Message**: Export with proper header formatting

**Expected:**
- Truncation works in all display modes
- No performance degradation
- Export includes full recipient lists (not truncated)

### Test Case 8: Performance Testing
**Input:**
- Email with 1000+ recipients
- Rapid switching between messages
- Multiple emails with long recipient lists

**Expected:**
- No noticeable lag in message display
- Memory usage remains reasonable
- UI remains responsive

### Performance Benchmarks:
- Header rendering: < 10ms for 100+ recipients
- Memory: No significant increase vs baseline
- UI responsiveness: No frame drops

## Manual Testing Checklist

### Pre-Implementation
- [ ] Create test emails with various recipient list lengths
- [ ] Document current behavior with screenshots
- [ ] Identify all code paths that render email headers

### During Implementation  
- [ ] Unit test truncation function with various inputs
- [ ] Test edge cases (empty recipients, malformed emails)
- [ ] Verify recipient counting accuracy
- [ ] Test different terminal widths

### Post-Implementation
- [ ] Visual verification in actual terminal
- [ ] Test with real Gmail data
- [ ] Performance testing with large recipient lists
- [ ] Accessibility testing (screen readers)
- [ ] Cross-platform testing (macOS, Linux)

### User Experience Testing
- [ ] Intuitive understanding of "... and X more recipients"
- [ ] Important fields (Labels) always visible
- [ ] No information loss for critical header data
- [ ] Consistent behavior across different email types

## Success Criteria

1. **Primary Goal**: Labels and other important fields always visible
2. **User Experience**: Clear indication of truncated recipients with count
3. **Performance**: No degradation in message display speed
4. **Reliability**: Handles edge cases gracefully (malformed recipients, very long addresses)
5. **Configurability**: Users can adjust truncation limits if needed
6. **Backward Compatibility**: No breaking changes to existing functionality

## Risk Mitigation

### Risk 1: Incorrect Recipient Counting
**Mitigation**: Comprehensive unit tests with various recipient formats

### Risk 2: UI Layout Issues
**Mitigation**: Test across multiple terminal sizes and themes

### Risk 3: Performance Degradation
**Mitigation**: Benchmark before/after, optimize truncation algorithm

### Risk 4: Information Loss
**Mitigation**: Preserve full recipient data for exports and advanced views

## Implementation Notes

- Keep existing `adjustHeaderHeight()` logic unchanged
- Truncation should be format-aware (handle quoted names, angle brackets)
- Consider future enhancement: press 'R' to show full recipient list
- Maintain compatibility with existing email export functionality
- Document configuration options in user documentation

## Future Enhancements

1. **Interactive Expansion**: Keypress to show full recipient list in popup
2. **Smart Truncation**: Show most important recipients first (sender domain priority)
3. **Visual Indicators**: Different styling for truncated vs full fields
4. **Recipient Grouping**: "5 recipients from @company.com, 3 others"