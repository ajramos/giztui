#!/bin/bash

# Architecture compliance checker for Gmail TUI
set -e

echo "🏗️ Checking architecture compliance..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

violations=0

# Check 1: No direct Gmail API calls in TUI components
echo "Checking for direct API calls in UI components..."
if grep -r "\.Client\." internal/tui/ --include="*.go" | grep -v "a\.Client" | grep -v "app\.Client" > /dev/null 2>&1; then
    echo -e "${RED}❌ Found direct API calls in TUI components:${NC}"
    grep -r "\.Client\." internal/tui/ --include="*.go" | grep -v "a\.Client" | grep -v "app\.Client"
    echo -e "${YELLOW}💡 Move API calls to services in internal/services/${NC}"
    violations=$((violations + 1))
fi

# Check 2: No fmt.Printf or log.Printf in TUI components (should use ErrorHandler)
echo "Checking for direct output in UI components..."
if grep -r -E "(fmt\.Printf|fmt\.Print|log\.Printf)" internal/tui/ --include="*.go" > /dev/null 2>&1; then
    echo -e "${RED}❌ Found direct output in TUI components:${NC}"
    grep -r -E "(fmt\.Printf|fmt\.Print|log\.Printf)" internal/tui/ --include="*.go"
    echo -e "${YELLOW}💡 Use app.GetErrorHandler().ShowError/ShowSuccess instead${NC}"
    violations=$((violations + 1))
fi

# Check 3: No direct field access (should use accessor methods)
echo "Checking for direct field access..."
if grep -r -E "a\.(currentView|currentMessageID|ids|running|screenWidth|screenHeight) *=" internal/tui/ --include="*.go" > /dev/null 2>&1; then
    echo -e "${RED}❌ Found direct field access in TUI components:${NC}"
    grep -r -E "a\.(currentView|currentMessageID|ids|running|screenWidth|screenHeight) *=" internal/tui/ --include="*.go"
    echo -e "${YELLOW}💡 Use thread-safe accessor methods like SetCurrentView() instead${NC}"
    violations=$((violations + 1))
fi

# Check 4: Services should implement interfaces
echo "Checking service interfaces..."
for service_file in internal/services/*_service.go; do
    if [ -f "$service_file" ]; then
        service_name=$(basename "$service_file" .go)
        interface_name=$(echo "$service_name" | sed 's/_service$/Service/' | sed 's/^./\U&/')
        
        if ! grep -q "type $interface_name interface" internal/services/interfaces.go; then
            echo -e "${YELLOW}⚠️ Service $service_name may be missing interface definition${NC}"
        fi
    fi
done

# Summary
if [ $violations -eq 0 ]; then
    echo -e "${GREEN}✅ All architecture checks passed!${NC}"
    exit 0
else
    echo -e "${RED}❌ Found $violations architecture violations${NC}"
    echo -e "${YELLOW}📖 See docs/ARCHITECTURE.md for guidance${NC}"
    exit 1
fi