// components/kanban/card-modal/useCardForm.ts
import { useState, useEffect, useCallback, useRef } from "react";
import type { Card, BoardMember, Tag } from "@/types/board";
import { API_URL } from "@/lib/constants";
import { FormState } from "../components/board/card-modal/CardDetailModal"; // หรือย้าย type FormState มาไว้ที่นี่

function buildInitialForm(card: Card): FormState {
  return {
    title: card.title,
    description: card.description ?? "",
    due_date: card.due_date ?? "",
    assignee_id: card.assignee_id ?? "",
    priority: card.priority ?? "",
    estimated_hours: card.estimated_hours != null ? String(card.estimated_hours) : "",
    tags: card.tags ?? [],
    acceptance_criteria: card.acceptance_criteria ?? "",
    implementation_note: card.implementation_note ?? "",
  };
}

// True when a single field differs between two form snapshots. Tags compare
// by id-set; everything else is a plain string/scalar.
function fieldEqual(field: keyof FormState, a: FormState, b: FormState): boolean {
  if (field === "tags") {
    const ai = a.tags.map((t) => t.id).sort().join(",");
    const bi = b.tags.map((t) => t.id).sort().join(",");
    return ai === bi;
  }
  return a[field] === b[field];
}

/**
 * Owns the editable form for one card. State initialises from the `card`
 * prop once and is NOT sync'd back via useEffect — callers must remount the
 * consuming modal with `key={card.id}` to reset the form when switching
 * cards. See CardDetailModal callers (BoardDashboard, TaskCard).
 *
 * Per-field auto-save (Option C): there is no batch "Save" button. Each field
 * commits on blur (text) or change (selects) by calling `commitField(field)`,
 * which fires `onCommit` with the full current form snapshot — the same shape
 * `handleUpdateCard` already consumes (it diffs changed_fields itself). We send
 * the whole snapshot rather than a partial so backend COALESCE never clobbers
 * an untouched field.
 */
export function useCardForm(
  card: Card,
  boardId: string,
  isOpen: boolean,
  onCommit?: (form: FormState) => void,
) {
  const [form, setForm] = useState<FormState>(() => buildInitialForm(card));

  // formRef mirrors `form` synchronously so a commit fired in the same event
  // as a change (e.g. picking a priority) reads the new value, not the stale
  // render closure. committedRef tracks the last value we actually persisted,
  // so a blur with no net change doesn't fire a redundant write.
  const formRef = useRef(form);
  const committedRef = useRef(form);

  const [members, setMembers] = useState<BoardMember[]>([]);
  const [error, setError] = useState<string | null>(null);

  // Single write path — keeps formRef current synchronously for callers that
  // commit immediately after a change.
  const updateForm = useCallback(<K extends keyof FormState>(field: K, value: FormState[K]) => {
    const next = { ...formRef.current, [field]: value } as FormState;
    formRef.current = next;
    setForm(next);
  }, []);

  // commitField closes over `card.title` / `onCommit` via deps rather than refs
  // so nothing is written to a ref during render (react-hooks/refs). The refs it
  // *reads* (formRef/committedRef) are only ever touched inside event handlers.
  const commitField = useCallback((field: keyof FormState) => {
    const cur = formRef.current;
    // Required-field guard: an empty title would 400 at the API. Revert to the
    // last good title and surface an inline error instead of saving.
    if (field === "title" && !cur.title.trim()) {
      const reverted = { ...cur, title: card.title };
      formRef.current = reverted;
      setForm(reverted);
      setError("Title cannot be empty.");
      return;
    }
    if (fieldEqual(field, cur, committedRef.current)) return;
    committedRef.current = { ...committedRef.current, [field]: cur[field] };
    setError(null);
    onCommit?.(cur);
  }, [card.title, onCommit]);

  // Fetch รายชื่อ Member ใน Board
  useEffect(() => {
    if (!isOpen || !boardId) return;
    const fetchMembers = async () => {
      try {
        const res = await fetch(`${API_URL}/boards/${boardId}/members`, {
          credentials: "include",
        });
        if (res.ok) setMembers(await res.json());
      } catch (err) {
        console.error("Failed to fetch members", err);
      }
    };
    fetchMembers();
  }, [isOpen, boardId]);

  // Helper สำหรับ Update State (text/select inputs).
  //
  // Each field's handler is cached so its identity is STABLE across renders.
  // Without this, `handleChange("title")` returns a fresh closure every render,
  // which defeats React.memo on every child receiving an onChange — the whole
  // modal then re-renders on every keystroke.
  type ChangeHandler = (
    e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>,
  ) => void;
  const handlersRef = useRef<Partial<Record<keyof FormState, ChangeHandler>>>({});
  const handleChange = useCallback((field: keyof FormState): ChangeHandler => {
    const cached = handlersRef.current[field];
    if (cached) return cached;
    const handler: ChangeHandler = (e) => updateForm(field, e.target.value);
    handlersRef.current[field] = handler;
    return handler;
  }, [updateForm]);

  // Setter สำหรับ tags (ไม่ผ่าน ChangeEvent เพราะเป็น array)
  const setTags = useCallback((tags: Tag[]) => {
    updateForm("tags", tags);
  }, [updateForm]);

  const assigneeName = members.find((m) => m.user_id === form.assignee_id)?.full_name;

  return {
    form,
    members,
    error,
    assigneeName,
    handleChange,
    setTags,
    commitField,
  };
}