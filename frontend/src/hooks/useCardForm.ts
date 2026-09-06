import { useState, useEffect, useCallback, useRef } from "react";
import type { Card, BoardMember, Tag } from "@/types/board";
import { API_URL } from "@/lib/constants";
import { FormState } from "../components/board/card-modal/CardDetailModal";

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

// True when a field differs between two snapshots; tags compare by id-set.
function fieldEqual(field: keyof FormState, a: FormState, b: FormState): boolean {
  if (field === "tags") {
    const ai = a.tags.map((t) => t.id).sort().join(",");
    const bi = b.tags.map((t) => t.id).sort().join(",");
    return ai === bi;
  }
  return a[field] === b[field];
}

/**
 * Owns the editable form for one card. State initialises from `card` once and is
 * never sync'd back — remount with `key={card.id}` to switch cards. Each field
 * commits the WHOLE snapshot, never a partial, or backend COALESCE clobbers
 * untouched fields.
 */
export function useCardForm(
  card: Card,
  boardId: string,
  isOpen: boolean,
  onCommit?: (form: FormState) => void,
) {
  const [form, setForm] = useState<FormState>(() => buildInitialForm(card));

  // formRef mirrors `form` synchronously so a commit in the same event as a change reads
  // the new value. committedRef holds the last persisted value, to skip no-op writes.
  const formRef = useRef(form);
  const committedRef = useRef(form);

  const [members, setMembers] = useState<BoardMember[]>([]);
  const [error, setError] = useState<string | null>(null);

  // Single write path — keeps formRef current synchronously for immediate commits.
  const updateForm = useCallback(<K extends keyof FormState>(field: K, value: FormState[K]) => {
    const next = { ...formRef.current, [field]: value } as FormState;
    formRef.current = next;
    setForm(next);
  }, []);

  // Deps, not refs, for card.title/onCommit — nothing may be written to a ref during
  // render (react-hooks/refs). formRef/committedRef are only read inside handlers.
  const commitField = useCallback((field: keyof FormState) => {
    const cur = formRef.current;
    // An empty title would 400 at the API; revert and surface an inline error.
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

  // Each field's handler is cached so its identity stays STABLE across renders —
  // a fresh closure per render defeats React.memo and re-renders the modal per keystroke.
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