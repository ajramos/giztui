# GizTUI Performance Test Plan
**Phase 1: Parallel Message Fetching + Phase 2.4: Background Preloading**

## 🎯 **Test Objectives**

### Primary Goals
- **Validate 5-10x performance improvement** in initial message loading (Phase 1)
- **Confirm instant navigation experience** through background preloading (Phase 2.4)
- **Ensure functional correctness** across all parallel loading scenarios  
- **Verify API rate limit compliance** and intelligent resource management
- **Confirm UI/UX consistency** with existing behavior patterns

### Success Criteria
- [ ] **Phase 1**: Inbox loading <2 seconds for 50 messages (vs 7+ seconds before)
- [ ] **Phase 1**: Remote search <3 seconds for search results
- [ ] **Phase 2.4**: Next page loads instantly (0.1-0.5s) after preloading
- [ ] **Phase 2.4**: Message navigation feels instant (0.1-0.3s) between preloaded messages
- [ ] **Phase 2.4**: Preloading triggers correctly at 70% scroll threshold
- [ ] Zero functional regressions in message display/interaction
- [ ] Graceful handling of API errors and intelligent resource limits

## 📊 **Performance Benchmarks**

### Measurement Methodology
```bash
# Time measurement points
1. Start: API call initiation  
2. First Response: First message received
3. Complete: All messages loaded and UI updated
4. Error Recovery: Time to handle/recover from failures
```

### Expected Performance Targets

| Scenario | Before (Sequential) | After (Parallel) | Target Improvement |
|----------|-------------------|------------------|-------------------|
| Inbox Load (50 msgs) | 7+ seconds | 1-2 seconds | 5-7x faster |
| Remote Search | 5-8 seconds | 1-3 seconds | 3-5x faster |
| Load More (50 msgs) | 6-7 seconds | 1-2 seconds | 4-6x faster |
| Error Recovery | Variable | <1 second | Consistent |

## 🧪 **Test Scenarios**

### **1. Functional Correctness Tests**

#### 1.1 Inbox Loading
- [ ] **Test**: Load inbox with 50 messages
- [ ] **Verify**: All messages display correctly with proper ordering
- [ ] **Check**: Message metadata (sender, subject, date, labels, attachments)
- [ ] **Validate**: Read/unread status indicators
- [ ] **Confirm**: Message numbering (if enabled)

#### 1.2 Remote Search Results  
- [ ] **Test**: Search with various Gmail query operators
  - Simple: `from:example@gmail.com`
  - Complex: `has:attachment after:2024/01/01 subject:invoice`
- [ ] **Verify**: Search results accuracy and completeness
- [ ] **Check**: System labels display correctly in search mode
- [ ] **Validate**: Message formatting with search context

#### 1.3 Load More Functionality
- [ ] **Test**: Load additional pages via "Load More"
- [ ] **Verify**: Seamless appending of new messages
- [ ] **Check**: No duplicate messages in list
- [ ] **Validate**: Pagination token handling
- [ ] **Confirm**: Focus restoration after loading

#### 1.4 Mixed Scenarios
- [ ] **Test**: Rapid switching between inbox/search/load more
- [ ] **Verify**: No race conditions or UI corruption
- [ ] **Check**: Proper cleanup of previous operations
- [ ] **Validate**: Consistent spinner/progress indicators

### **2. Performance Tests**

#### 2.1 Load Time Measurements
```bash
# Test commands for manual verification
1. Fresh inbox load: time ./build/giztui 
2. Search performance: measure search response time
3. Load more timing: measure pagination response time
```

#### 2.2 Concurrent Load Testing
- [ ] **Test**: Multiple rapid refresh operations  
- [ ] **Verify**: System handles concurrent requests gracefully
- [ ] **Check**: No memory leaks during extended use
- [ ] **Validate**: API rate limit compliance

#### 2.3 Network Condition Tests
- [ ] **Test**: Performance under various network conditions
  - Fast connection (fiber/5G)
  - Slow connection (3G simulation) 
  - Intermittent connectivity
- [ ] **Verify**: Graceful degradation and error handling

### **3. Error Handling Tests**

#### 3.1 API Failure Scenarios
- [ ] **Test**: Individual message fetch failures (404, 403, etc.)
- [ ] **Verify**: Failed messages show error indicators
- [ ] **Check**: Successful messages still display correctly
- [ ] **Validate**: No complete operation failure from partial errors

#### 3.2 Rate Limiting Tests
- [ ] **Test**: Rapid consecutive API calls to trigger rate limits
- [ ] **Verify**: Graceful handling without crashes
- [ ] **Check**: Appropriate error messages to user
- [ ] **Validate**: Automatic retry mechanisms (if implemented)

#### 3.3 Network Error Recovery
- [ ] **Test**: Network disconnection during message loading
- [ ] **Verify**: Clear error messages and recovery options
- [ ] **Check**: UI remains responsive and stable
- [ ] **Validate**: Retry functionality works correctly

### **4. Integration Tests**

#### 4.1 Existing Feature Compatibility
- [ ] **Test**: Message operations (archive, delete, label, etc.)
- [ ] **Verify**: All keyboard shortcuts work correctly
- [ ] **Check**: Theme system applies correctly to loaded messages
- [ ] **Validate**: Picker panels (labels, attachments, etc.) function

#### 4.2 Threading Mode Compatibility
- [ ] **Test**: Thread view with parallel loading
- [ ] **Verify**: Thread expansion/collapse works correctly
- [ ] **Check**: Thread message ordering maintained
- [ ] **Validate**: Threading indicators display properly

### **5. Phase 2.4: Background Preloading Tests**

#### 5.1 Next Page Preloading
- [ ] **Test**: Scroll to 70% of message list (default threshold)
- [ ] **Verify**: Next page preloading triggers automatically in background
- [ ] **Check**: No UI blocking or spinner during preloading
- [ ] **Validate**: "Load More" button responds instantly (<0.5s) when clicked
- [ ] **Test**: Adjust threshold to 50% via `:preload next threshold 0.5`
- [ ] **Verify**: Preloading triggers at new threshold

#### 5.2 Adjacent Message Preloading  
- [ ] **Test**: Navigate through message list with arrow keys
- [ ] **Verify**: Messages around selection load instantly (<0.3s)
- [ ] **Check**: No delay when moving between preloaded messages
- [ ] **Validate**: Preloading count configurable (default: 3 messages)
- [ ] **Test**: Rapid navigation through 10+ messages
- [ ] **Verify**: Smooth experience with no loading delays

#### 5.3 Preloading Configuration Tests
- [ ] **Test**: `:preload status` command shows accurate statistics
  - Cache hit rates, memory usage, worker status
- [ ] **Test**: `:preload off` disables all background preloading
- [ ] **Verify**: No background activity after disabling
- [ ] **Test**: `:preload on` re-enables preloading
- [ ] **Test**: `:preload clear` empties all caches
- [ ] **Verify**: Memory usage drops after cache clear

#### 5.4 Resource Management Tests
- [ ] **Test**: Preloading respects background_workers limit (default: 3)
- [ ] **Verify**: No more than 3 concurrent background requests
- [ ] **Test**: Cache size limit prevents excessive memory usage (default: 50MB)
- [ ] **Verify**: LRU eviction works when cache fills up
- [ ] **Test**: API quota reserve (default: 20%) prevents interactive blocking
- [ ] **Verify**: Preloading pauses when quota threshold reached

#### 5.5 Preloading Behavior Tests
- [ ] **Test**: Switch between different views (inbox, search results, labels)
- [ ] **Verify**: Preloading adapts to current context
- [ ] **Check**: Cache doesn't interfere between different message lists
- [ ] **Test**: Long session with extensive navigation
- [ ] **Verify**: No memory leaks or cache overflow
- [ ] **Validate**: Performance remains consistent over time

#### 5.6 Runtime Configuration Tests
- [ ] **Test**: Change preloading settings during active session
- [ ] **Verify**: Configuration changes take effect immediately
- [ ] **Test**: `:preload next max_pages 1` reduces preloading scope
- [ ] **Test**: `:preload adjacent count 5` increases preloading range  
- [ ] **Verify**: Modified settings persist correctly

## 🛠️ **Test Execution Plan**

### Phase 1: Automated Testing (If Available)
```bash
# Run existing test suite
make test-all

# Performance benchmarks  
make test-performance  # (if available)

# Integration tests
make test-integration
```

### Phase 2: Manual Testing Checklist

#### Setup Requirements
- [ ] Gmail account with 50+ messages in inbox
- [ ] Various message types (attachments, labels, read/unread)
- [ ] Network monitoring tools for performance measurement
- [ ] Multiple terminal sessions for concurrent testing

#### Test Execution Order
1. **Baseline Measurement** (on main branch)
   - [ ] Record current performance metrics
   - [ ] Document existing behavior patterns

2. **Feature Branch Testing** (performance/parallel-message-loading)
   - [ ] Execute all Phase 1 parallel loading tests
   - [ ] Execute all Phase 2.4 background preloading tests
   - [ ] Measure and record performance improvements  
   - [ ] Test error handling scenarios
   - [ ] Validate integration with existing features
   - [ ] Test runtime configuration and command interface

3. **Regression Testing**
   - [ ] Compare feature branch vs main branch
   - [ ] Identify any behavioral changes
   - [ ] Ensure no functionality lost

### Phase 3: User Acceptance Testing

#### Real-World Usage Scenarios
- [ ] **Morning email check**: Load inbox, navigate through messages (test preloading)
- [ ] **Email search workflow**: Search, refine, browse results (test adaptive preloading)
- [ ] **Extended session**: Browse multiple pages, test "Load More" instant response
- [ ] **Power user workflow**: Rapid navigation with preload commands (`:preload status`)
- [ ] **Multi-tasking**: Email handling while other apps running (test resource limits)

## 📈 **Performance Monitoring**

### Metrics to Collect
```go
// Add timing measurements to code (temporary for testing)
start := time.Now()
messages, err := a.Client.GetMessagesParallel(messageIDs, 10)
loadTime := time.Since(start)
log.Printf("Parallel fetch took: %v for %d messages", loadTime, len(messageIDs))
```

### Success Indicators
- [ ] **Speed**: Consistent 1-2 second load times for 50 messages
- [ ] **Reliability**: <1% message fetch failure rate
- [ ] **Stability**: No memory leaks during 30-minute test sessions
- [ ] **UX**: Smooth, responsive interface throughout testing

## 🔍 **Known Risks & Mitigation**

### Potential Issues
1. **API Rate Limiting**: Gmail API quotas exceeded
   - **Mitigation**: Conservative worker pool size (10 workers)
   - **Test**: Gradual load increase to find limits

2. **Memory Usage**: Concurrent operations increase memory
   - **Mitigation**: Monitor memory during extended testing
   - **Test**: Long-running sessions with multiple operations

3. **Error Handling**: Partial failures in parallel operations
   - **Mitigation**: Robust error handling per message
   - **Test**: Simulate various API failure scenarios

## 📝 **Test Results Documentation**

### Results Template
```markdown
## Test Results - [Date]

### Performance Metrics
- Inbox Load Time: X.X seconds (vs X.X before)
- Search Response Time: X.X seconds  
- Load More Time: X.X seconds
- Overall Improvement: Xx faster

### Functional Testing
- ✅/❌ All messages display correctly
- ✅/❌ No regressions in existing features  
- ✅/❌ Error handling works as expected

### Issues Found
- [List any bugs or regressions discovered]

### Recommendations
- [Suggestions for improvements or fixes needed]
```

---

**Test Execution Status**: Ready for implementation
**Next Steps**: Execute test plan and document results before merge to main