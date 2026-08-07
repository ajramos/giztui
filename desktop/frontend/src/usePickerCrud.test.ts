import { describe, it, expect } from "vitest";
import { pickerCrudAction } from "./usePickerCrud";

describe("pickerCrudAction", () => {
  it("maps bare keys to actions in list mode (not typing)", () => {
    expect(pickerCrudAction("e", false)).toBe("edit");
    expect(pickerCrudAction("E", false)).toBe("edit");
    expect(pickerCrudAction("d", false)).toBe("delete");
    expect(pickerCrudAction("Delete", false)).toBe("delete");
    expect(pickerCrudAction("Backspace", false)).toBe("delete");
  });

  it("does nothing while a text field is focused (every key is typing)", () => {
    expect(pickerCrudAction("e", true)).toBeNull();
    expect(pickerCrudAction("E", true)).toBeNull();
    expect(pickerCrudAction("d", true)).toBeNull();
    expect(pickerCrudAction("Delete", true)).toBeNull();
    expect(pickerCrudAction("Backspace", true)).toBeNull();
  });

  it("returns null for unrelated keys", () => {
    expect(pickerCrudAction("ArrowDown", false)).toBeNull();
    expect(pickerCrudAction("Enter", false)).toBeNull();
    expect(pickerCrudAction("x", false)).toBeNull();
    expect(pickerCrudAction("n", false)).toBeNull();
  });
});
