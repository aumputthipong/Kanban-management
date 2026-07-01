import { describe, it, expect, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { PriorityFilterDropdown } from "@/components/board/task-board/PriorityFilterDropdown";
import { useBoardStore } from "@/store/useBoardStore";

beforeEach(() => {
  useBoardStore.setState({ filterPriorities: [] });
});

describe("PriorityFilterDropdown", () => {
  it("renders the trigger button", () => {
    render(<PriorityFilterDropdown />);
    expect(screen.getByText("Priority")).toBeInTheDocument();
  });

  it("dropdown is hidden initially", () => {
    render(<PriorityFilterDropdown />);
    expect(screen.queryByText("High")).not.toBeInTheDocument();
    expect(screen.queryByText("Medium")).not.toBeInTheDocument();
    expect(screen.queryByText("Low")).not.toBeInTheDocument();
  });

  it("opens dropdown when trigger is clicked", async () => {
    render(<PriorityFilterDropdown />);
    await userEvent.click(screen.getByText("Priority"));

    expect(screen.getByText("High")).toBeInTheDocument();
    expect(screen.getByText("Medium")).toBeInTheDocument();
    expect(screen.getByText("Low")).toBeInTheDocument();
  });

  it("toggles a priority on click and updates store", async () => {
    render(<PriorityFilterDropdown />);
    await userEvent.click(screen.getByText("Priority"));
    await userEvent.click(screen.getByText("High"));

    expect(useBoardStore.getState().filterPriorities).toContain("high");
  });

  it("shows active count badge when priorities are selected", async () => {
    render(<PriorityFilterDropdown />);
    await userEvent.click(screen.getByText("Priority"));
    await userEvent.click(screen.getByText("High"));
    await userEvent.click(screen.getByText("Medium"));

    // badge shows the selected count
    expect(screen.getByText("2")).toBeInTheDocument();
    expect(useBoardStore.getState().filterPriorities).toHaveLength(2);
  });

  it("removes a priority when clicking it again (toggle off)", async () => {
    render(<PriorityFilterDropdown />);
    await userEvent.click(screen.getByText("Priority"));
    await userEvent.click(screen.getByText("High"));
    await userEvent.click(screen.getByText("High")); // toggle off

    expect(useBoardStore.getState().filterPriorities).not.toContain("high");
  });

  it("shows 'Clear filter' button when a priority is active", async () => {
    render(<PriorityFilterDropdown />);
    await userEvent.click(screen.getByText("Priority"));
    await userEvent.click(screen.getByText("Low"));

    expect(screen.getByText("Clear filter")).toBeInTheDocument();
  });

  it("clears all priorities and closes dropdown when 'Clear filter' is clicked", async () => {
    render(<PriorityFilterDropdown />);
    await userEvent.click(screen.getByText("Priority"));
    await userEvent.click(screen.getByText("High"));
    await userEvent.click(screen.getByText("Clear filter"));

    expect(useBoardStore.getState().filterPriorities).toHaveLength(0);
    // dropdown closes after clear
    expect(screen.queryByText("High")).not.toBeInTheDocument();
  });
});
