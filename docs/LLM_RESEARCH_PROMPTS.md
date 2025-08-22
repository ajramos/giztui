# LLM Research Prompts

This document contains detailed research prompts for investigating complex technical issues that require deep analysis.

## tview Color Tag Processing in Theme Preview Research

### Background Context
**Issue**: Theme preview in Theme Picker displays tview color tags as literal text instead of rendering them as colored text.

**Environment**:
- Framework: `github.com/derailed/tview v0.8.5` 
- Terminal: `github.com/derailed/tcell/v2 v2.3.1-rc.4`
- Language: Go
- Application: Terminal-based email client with dynamic theme system

### Research Prompt

```
I need you to conduct comprehensive research into a tview framework color tag processing issue. Here's the detailed problem:

## Problem Statement
In a Go application using the tview TUI framework, we're experiencing color tags not being processed in theme preview context:

1. **Expected behavior**: Color tags like `[#ffb86c]Primary Color[-]` should render as colored text
2. **Actual behavior**: Color tags display as literal text: "[#ffb86c]Primary Color[-]"
3. **Context**: This occurs specifically in theme preview functionality within a Theme Picker component

The issue affects user experience by showing raw color codes instead of visual color samples.

## Technical Details
- **Library Version**: github.com/derailed/tview v0.8.5 (fork of rivo/tview)
- **Terminal Library**: github.com/derailed/tcell/v2 v2.3.1-rc.4
- **Component Type**: TextView with SetDynamicColors(true) enabled
- **Theme System**: Dynamic theme loading that updates tview.Styles.* values at runtime
- **Content Generation**: Programmatically generated text with color tags

## Code Context & UI Hierarchy Integration

### Theme Preview TextView Integration:
**CRITICAL CONTEXT**: The theme preview reuses the existing main message content TextView instead of creating a fresh TextView. Here's the exact integration:

1. **UI Hierarchy**: 
   ```
   main -> Pages -> textContainer (Flex) -> text (TextView)
   ```

2. **TextView Reuse Pattern**:
   ```go
   // In showThemePreview() - reuses existing TextView
   a.QueueUpdateDraw(func() {
       if textView, ok := a.views["text"].(*tview.TextView); ok {
           textView.SetText(details) // Overwrites existing message content
           textView.ScrollToBeginning()
       }
   })
   ```

3. **TextView Initialization** (from layout.go):
   ```go
   enhancedText := NewEnhancedTextView(a)
   text := enhancedText.TextView
   text.SetDynamicColors(true).SetWrap(true).SetScrollable(true)
   text.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
   a.views["text"] = text // Stored for reuse
   ```

4. **Container Hierarchy**:
   ```go
   textContainer := tview.NewFlex().SetDirection(tview.FlexRow)
   textContainer.SetBorder(true)
   textContainer.AddItem(header, 6, 0, false)
   textContainer.AddItem(text, 0, 1, false) // TextView embedded in Flex
   ```

5. **Color Sample Generation**:
   ```go
   func (a *App) formatColorSampleString(name string, colorValue string) string {
       namedColor := a.hexToNamedColor(colorValue)
       if namedColor != "" {
           return fmt.Sprintf("  [%s]●[-] [%s]%s[-] (%s)\n", namedColor, namedColor, name, colorValue)
       }
       return fmt.Sprintf("  ● %s (%s)\n", name, colorValue) // Fallback without color tags
   }
   ```

**Key Integration Details**:
- **Reused TextView**: Same TextView that displays email content is reused for theme preview
- **Embedded in Flex**: TextView is inside a bordered Flex container (textContainer)
- **Dynamic Content**: Color-tagged content is generated programmatically and set via SetText()
- **Container Manipulation**: Theme preview modifies the parent container's title during preview
- **Focus Management**: Preview focuses the TextView/EnhancedTextView for scrolling

## Failed Approaches
Multiple solutions were attempted:

1. **Hex-to-named color mapping**: Converted hex values (#ffb86c) to tview named colors (orange)
2. **Dynamic color verification**: Confirmed SetDynamicColors(true) was properly configured
3. **Color tag syntax testing**: Tested both hex `[#ffb86c]` and named `[orange]` formats
4. **TextView attribute validation**: Ensured proper setup of all color processing attributes
5. **Content formatting variations**: Tried different string generation approaches

**Result**: None achieved color tag processing in the theme preview context.

## Research Objectives
Please investigate and provide detailed analysis on:

### 1. Root Cause Analysis
- Examine tview TextView color tag processing requirements and limitations
- Identify specific conditions required for dynamic color rendering to work
- Analyze why color tags work in some tview contexts but not others
- Investigate terminal-specific rendering differences that might affect color processing

### 2. Technical Deep Dive
- How does tview handle color tag parsing and rendering internally?
- Are there TextView initialization parameters or state that affects color processing?
- What role does the parent container or layout context play in color rendering?
- Are there undocumented methods or configuration options for color processing?

### 3. Context-Specific Investigation
**SPECIFIC TO OUR INTEGRATION**:
- Why don't color tags work when TextView is reused and content is set dynamically via SetText()?
- Does embedding TextView inside a Flex container affect color tag processing?
- Are there state issues when the same TextView transitions from message content to theme preview?
- Does the QueueUpdateDraw() wrapper affect color tag parsing timing?
- Is there something about the EnhancedTextView wrapper that interferes with color processing?
- Do container title changes or focus management affect color rendering?

**GENERAL INVESTIGATION**:
- Why do color tags work in other parts of the application but not theme preview?
- Is there something special about dynamic content generation vs static content?
- Do nested containers or complex layouts interfere with color processing?
- Are there timing issues with when color tags are parsed vs when content is rendered?

### 4. Community Solutions & Patterns
- Search for similar issues in tview/rivo communities (GitHub issues, discussions)
- Look for successful implementations of color tags in similar contexts
- Check for known workarounds or best practices for dynamic color content
- Identify if this is a documented limitation or undiscovered bug

### 5. Alternative Implementation Strategies
- Suggest technical approaches that don't rely on color tag processing
- Evaluate feasibility of custom color rendering implementations
- Consider alternative UI patterns for theme preview that would be more reliable
- Propose minimal changes to achieve visual color representation

### 6. Version and Compatibility Analysis
- Check if this issue exists across different tview versions (rivo/tview vs derailed/tview)
- Identify any recent commits or PRs related to color tag processing
- Analyze changelog entries for color-related fixes or changes
- Compare behavior with different terminal emulators and color capabilities

## Expected Deliverables
1. **Detailed technical explanation** of why color tags don't process in this context
2. **Specific code references** from tview source showing the color processing logic
3. **Ranked list of solution approaches** with feasibility assessment
4. **Proof-of-concept code** for the most promising alternative approach
5. **Testing strategy** to validate any proposed fixes across different terminals

## Specific Code Areas to Investigate
The issue occurs in theme preview generation within:
- Theme picker component with dynamic content
- TextView components with programmatically set content
- Complex nested layout structures
- Runtime theme switching functionality

## Success Criteria
An ideal solution would:
- Enable visual color representation in theme previews
- Work reliably across different terminal environments  
- Not require major architectural changes to the theme system
- Maintain compatibility with existing tview patterns
- Be maintainable and well-documented

Focus your research on understanding the fundamental differences in how tview processes color tags in different contexts, and provide actionable solutions that could be implemented without extensive framework modifications.
```

### Usage Instructions
1. Copy the research prompt above to your preferred LLM (Claude, GPT-4, etc.)
2. Include relevant tview documentation or source code if available
3. Request follow-up analysis on specific technical findings
4. Validate any proposed solutions against the application's theme system requirements

### Research Notes
- Focus on the `github.com/derailed/tview` fork specifically, as behavior may differ from `rivo/tview`
- Consider that the theme system dynamically updates `tview.Styles.*` values at runtime
- Look for TextView lifecycle methods that could be leveraged for color processing
- Investigate if the issue is related to the timing of when color tags are parsed vs content rendering

---

*Last Updated: August 22, 2025*
*Related Issue: See KNOWN_ISSUES.md for complete problem documentation*