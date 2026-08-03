import { describe, it, expect } from "vitest";
import { pickerCrudAction } from "./usePickerCrud";

describe("pickerCrudAction", () => {
  it("maps bare keys to actions when not typing", () => {
    expect(pickerCrudAction("e", false, false)).toBe("edit");
    expect(pickerCrudAction("d", false, false)).toBe("delete");
    expect(pickerCrudAction("Delete", false, false)).toBe("delete");
    expect(pickerCrudAction("Backspace", false, false)).toBe("delete");
  });

  it("ignores bare keys while typing in a text field", () => {
    expect(pickerCrudAction("e", false, true)).toBeNull();
    expect(pickerCrudAction("d", false, true)).toBeNull();
    expect(pickerCrudAction("Backspace", false, true)).toBeNull();
  });

  it("honours Shift variants even while typing", () => {
    expect(pickerCrudAction("E", true, true)).toBe("edit");
    expect(pickerCrudAction("Delete", true, true)).toBe("delete");
    expect(pickerCrudAction("Backspace", true, true)).toBe("delete");
  });

  it("Shift variants also act when not typing", () => {
    expect(pickerCrudAction("E", true, false)).toBe("edit");
    expect(pickerCrudAction("Delete", true, false)).toBe("delete");
  });

  it("returns null for unrelated keys", () => {
    expect(pickerCrudAction("ArrowDown", false, false)).toBeNull();
    expect(pickerCrudAction("Enter", false, false)).toBeNull();
    expect(pickerCrudAction("x", false, false)).toBeNull();
    expect(pickerCrudAction("n", false, false)).toBeNull();
  });
});
