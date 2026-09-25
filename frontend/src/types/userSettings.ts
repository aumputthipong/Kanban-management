export type DefaultLanding = "today" | "my_work" | "all_boards";

export interface UserSettings {
  default_landing: DefaultLanding;
  show_all_cards: boolean;
  timezone: string;
}

export interface UpdateUserSettingsBody {
  default_landing?: DefaultLanding;
  show_all_cards?: boolean;
  timezone?: string;
}

export function isDefaultLanding(value: string): value is DefaultLanding {
  return value === "today" || value === "my_work" || value === "all_boards";
}

// "today" was merged into My Work — both land on /my-work.
export const LANDING_PATH: Record<DefaultLanding, string> = {
  today: "/my-work",
  my_work: "/my-work",
  all_boards: "/dashboard",
};
