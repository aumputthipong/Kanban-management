import type { BoardMember, Card, CardUpdateForm } from "@/types/board";

export type CardField = keyof CardUpdateForm;

export type CardPatchBody = {
  title?: string;
  description?: string;
  due_date?: string;
  assignee_id?: string;
  priority?: string;
  estimated_hours?: number;
  tag_ids?: string[];
  acceptance_criteria?: string;
  implementation_note?: string;
  changed_fields: string[];
};

/**
 * Builds the PATCH body and optimistic store patch for ONE committed field. Sending
 * more lets a stale modal overwrite someone else's edit; `null` cannot clear, so
 * clears go as "" or 0 (docs/adr/0009-card-patch-sends-changed-fields.md).
 */
export function buildCardFieldUpdate(
  form: CardUpdateForm,
  field: CardField,
  members: BoardMember[],
): { body: CardPatchBody; patch: Partial<Card> } {
  const changed_fields = [field];

  switch (field) {
    case "title":
      return { body: { title: form.title, changed_fields }, patch: { title: form.title } };
    case "description":
    case "due_date":
    case "priority":
    case "acceptance_criteria":
    case "implementation_note": {
      const value = form[field];
      return {
        body: { [field]: value, changed_fields },
        patch: { [field]: value || null } as Partial<Card>,
      };
    }
    case "assignee_id": {
      const id = form.assignee_id;
      const name = id ? (members.find((m) => m.user_id === id)?.full_name ?? null) : null;
      return {
        body: { assignee_id: id, changed_fields },
        patch: { assignee_id: id || null, assignee_name: name },
      };
    }
    case "estimated_hours": {
      const parsed = parseFloat(form.estimated_hours);
      const hours = Number.isFinite(parsed) && parsed > 0 ? parsed : null;
      return {
        body: { estimated_hours: hours ?? 0, changed_fields },
        patch: { estimated_hours: hours },
      };
    }
    case "tags":
      return {
        body: { tag_ids: form.tags.map((t) => t.id), changed_fields },
        patch: { tags: form.tags },
      };
  }
}
