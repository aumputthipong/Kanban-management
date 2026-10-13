import { describe, it, expect } from "vitest";
import {
  atLeast,
  canDeleteCard,
  canEditCard,
  canManageColumns,
  canManageMember,
  grantableRoles,
  type BoardRole,
} from "@/lib/permissions";

const roles: BoardRole[] = ["owner", "manager", "member"];

describe("atLeast", () => {
  it("ranks owner > manager > member and rejects no role", () => {
    expect(atLeast("owner", "manager")).toBe(true);
    expect(atLeast("member", "manager")).toBe(false);
    expect(atLeast(null, "member")).toBe(false);
  });
});

describe("card rules", () => {
  it("lets the assignee edit but not delete", () => {
    expect(canEditCard("member", false, true)).toBe(true);
    expect(canDeleteCard("member", false)).toBe(false);
  });

  it("lets the creator and managers delete", () => {
    expect(canDeleteCard("member", true)).toBe(true);
    expect(canDeleteCard("manager", false)).toBe(true);
  });
});

describe("canManageColumns", () => {
  it.each([
    ["owner", true],
    ["manager", true],
    ["member", false],
  ] as const)("%s -> %s", (role, want) => {
    expect(canManageColumns(role)).toBe(want);
  });
});

describe("member rules", () => {
  it("lets managers act on members only", () => {
    expect(roles.filter((t) => canManageMember("manager", t))).toEqual(["member"]);
  });

  it("lets the owner act on everyone but the owner", () => {
    expect(roles.filter((t) => canManageMember("owner", t))).toEqual(["manager", "member"]);
  });

  it("never lets a member act on anyone", () => {
    expect(roles.filter((t) => canManageMember("member", t))).toEqual([]);
  });

  it("never offers owner as a grantable role", () => {
    expect(grantableRoles("owner")).toEqual(["manager", "member"]);
    expect(grantableRoles("manager")).toEqual(["member"]);
    expect(grantableRoles("member")).toEqual([]);
  });
});
