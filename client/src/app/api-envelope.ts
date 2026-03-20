/** Standard success envelope from GET/POST/PUT JSON responses. */
export interface ApiEnvelope<T> {
  data: T;
  meta: { request_id: string };
}

export interface TokenResponseData {
  access_token: string;
  token_type: string;
  expires_in: number;
}
