# LLM Research Prompts

This document contains detailed research prompts for investigating complex technical issues that require deep analysis.

## tview Border Rendering Inconsistency Research

### Background Context
**Issue**: Different tview component types (Table vs Flex) render borders with inconsistent visual appearance despite identical styling configuration.

**Environment**:
- Framework: `github.com/derailed/tview v0.8.5` 
- Terminal: `github.com/derailed/tcell/v2 v2.3.1-rc.4`
- Language: Go
- Application: Terminal-based email client with complex UI layouts

### Research Prompt

```
I need you to conduct deep research into a tview framework border rendering inconsistency. Here's the detailed problem:

## Problem Statement
In a Go application using the tview TUI framework, we're experiencing inconsistent border rendering between different component types:

1. **tview.Table components**: Borders appear "filled/solid" - the border area has the same color as the component background
2. **tview.Flex components**: Borders appear "hollow/transparent" - the border area shows through to underlying backgrounds

Both component types use identical styling:
```go
component.SetBorder(true).
    SetBorderColor(tview.Styles.PrimitiveBackgroundColor).
    SetBorderAttributes(tcell.AttrBold)
```

## Technical Details
- **Library Version**: github.com/derailed/tview v0.8.5 (fork of rivo/tview)
- **Terminal Library**: github.com/derailed/tcell/v2 v2.3.1-rc.4
- **Configuration**: All components use `tview.Styles.PrimitiveBackgroundColor` for border color
- **Theme System**: Dynamic theme loading that updates `tview.Styles.*` values at runtime

## Failed Approaches
We attempted multiple solutions:
1. Component-level styling modifications (different border colors, attributes)
2. Wrapper container hierarchies for background inheritance
3. Root-level background application
4. Theme YAML file modifications
5. Global tview.Styles overrides at runtime

None achieved consistent border appearance between Table and Flex components.

## Research Objectives
Please investigate and provide detailed analysis on:

### 1. Root Cause Analysis
- Examine tview source code differences between Table and Flex border rendering
- Identify specific code paths that cause different visual behavior
- Analyze how SetBorderColor() is implemented differently across component types
- Investigate terminal rendering differences (tcell layer interactions)

### 2. Technical Deep Dive
- How does tview handle border drawing at the tcell level?
- Are there component-specific override methods for border rendering?
- What role does the component's internal layout system play in border appearance?
- Are there undocumented styling methods or properties that could help?

### 3. Community Solutions
- Search for similar issues in tview/rivo communities (GitHub issues, discussions)
- Look for forks or patches that address border rendering inconsistencies
- Check for documented workarounds or best practices
- Identify if this is a known limitation or bug

### 4. Alternative Approaches
- Suggest technical workarounds that don't require library modification
- Evaluate feasibility of custom border drawing implementations
- Consider component architecture changes that could unify appearance
- Propose minimal patches to the tview library if needed

### 5. Version Analysis
- Check if this issue exists in different tview versions (rivo/tview vs derailed/tview)
- Identify any recent commits or PRs related to border rendering
- Analyze changelog entries for border-related fixes or changes

## Expected Deliverables
1. **Detailed technical explanation** of why this inconsistency occurs
2. **Specific code references** from tview source showing the differences
3. **Ranked list of solution approaches** with complexity/risk assessment
4. **Proof-of-concept code** for the most promising solution
5. **Testing strategy** to validate any proposed fixes

## Code Context
The application uses a complex layout with:
- Message list (Table) with per-row styling
- Message content area (Flex) with nested components  
- Theme picker (Flex) added dynamically to content splits
- Global theme system that updates all styling dynamically

Focus your research on understanding the fundamental differences in how these component types handle the visual rendering of their border areas, particularly when border color matches component background color.
```

### Usage Instructions
1. Copy the research prompt above to your preferred LLM (Claude, GPT-4, etc.)
2. Include relevant code snippets or tview documentation if available
3. Request follow-up analysis on specific findings
4. Validate any proposed solutions against the application requirements

### Research Notes
- Focus on the `github.com/derailed/tview` fork specifically, as it may have different behavior than `rivo/tview`
- Consider terminal-specific rendering differences that might affect border appearance
- Look for component lifecycle methods that could be overridden for custom border handling
- Investigate if the issue is related to the complex nested layout structure used in the application

---

*Last Updated: August 22, 2025*
*Related Issue: See KNOWN_ISSUES.md for complete problem documentation*