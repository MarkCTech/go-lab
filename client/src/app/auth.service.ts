import { Injectable } from '@angular/core';
import { HttpClient, HttpErrorResponse } from '@angular/common/http';
import { Observable, of } from 'rxjs';
import { catchError, map, tap } from 'rxjs/operators';
import { environment } from '../environments/environment';
import { ApiEnvelope, TokenResponseData } from './api-envelope';

@Injectable({
  providedIn: 'root'
})
export class AuthService {
  private accessToken: string | null = null;

  constructor(private http: HttpClient) {}

  /** Load API bootstrap token without storing credentials in the SPA bundle. */
  loadToken(): Observable<void> {
    return this.http
      .post<ApiEnvelope<TokenResponseData>>(
        `${environment.apiBaseUrl}/api/v1/auth/bootstrap`,
        {}
      )
      .pipe(
        tap((res) => {
          this.accessToken = res.data.access_token;
        }),
        map(() => undefined),
        catchError((err) => {
          if (err instanceof HttpErrorResponse) {
            console.error(
              `AuthService: token request failed ${err.status} ${err.statusText}`,
              err.error
            );
          } else {
            console.error('AuthService: token request failed', err);
          }
          return of(undefined);
        })
      );
  }

  getAccessToken(): string | null {
    return this.accessToken;
  }
}
