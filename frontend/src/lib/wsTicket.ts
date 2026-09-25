import { apiClient } from "@/lib/apiClient";

interface WsTicketResponse {
  ticket: string;
  expires_in: number;
}

/** Fetch a fresh ticket per connection attempt — it expires in seconds (docs/adr/0005). */
export async function fetchWsTicket(): Promise<string> {
  const { ticket } = await apiClient<WsTicketResponse>("/ws-ticket");
  return ticket;
}
