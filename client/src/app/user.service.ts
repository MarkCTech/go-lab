import { Injectable } from '@angular/core';
import { User } from './user';
import { Observable, of } from 'rxjs';
import { catchError, map, tap } from 'rxjs/operators';
import { MessageService } from './message.service';
import {
  HttpClient,
  HttpErrorResponse,
  HttpHeaders,
  HttpResponse
} from '@angular/common/http';
import { environment } from '../environments/environment';
import { ApiEnvelope } from './api-envelope';

@Injectable({
  providedIn: 'root'
})
export class UserService {
  private usersUrl = `${environment.apiBaseUrl}/api/v1/users`;

  constructor(
    private http: HttpClient,
    private messageService: MessageService
  ) {}

  getUsers(): Observable<User[]> {
    return this.http
      .get<ApiEnvelope<User[]>>(this.usersUrl)
      .pipe(
        map((e) => e.data),
        tap(() => this.log('Fetched users')),
        catchError(this.handleError<User[]>('getUsers', []))
      );
  }

  getUser(id: number): Observable<User> {
    const url = `${this.usersUrl}/${id}`;
    return this.http.get<ApiEnvelope<User>>(url).pipe(
      map((e) => e.data),
      tap((u) => {
        const outcome = u ? 'Fetched' : 'Did not find';
        this.log(`${outcome} user id=${id}`);
      }),
      catchError(this.handleError<User>(`getUser id=${id}`))
    );
  }

  searchUsers(term: string): Observable<User[]> {
    if (!term.trim()) {
      return of([]);
    }
    return this.http
      .get<ApiEnvelope<User[]>>(
        `${this.usersUrl}/search?name=${encodeURIComponent(term)}`
      )
      .pipe(
        map((e) => e.data),
        tap((x) =>
          x.length
            ? this.log(`Found users matching "${term}"`)
            : this.log(`No users matching "${term}"`)
        ),
        catchError(this.handleError<User[]>(`searchUsers`, []))
      );
  }

  addUser(user: User): Observable<User> {
    return this.http
      .post<ApiEnvelope<User>>(this.usersUrl, user, this.httpOptions)
      .pipe(
        map((e) => e.data),
        tap((newuser) => this.log(`Added user w/ id=${newuser.id}`)),
        catchError(this.handleError<User>('addUser'))
      );
  }

  updateUser(user: User): Observable<User> {
    return this.http
      .put<ApiEnvelope<User>>(
        `${this.usersUrl}/${user.id}`,
        user,
        this.httpOptions
      )
      .pipe(
        map((e) => e.data),
        tap(() => this.log(`Updated user id=${user.id}`)),
        catchError(this.handleError<User>('updateUser'))
      );
  }

  deleteUser(id: number): Observable<boolean> {
    const url = `${this.usersUrl}/${id}`;
    return this.http
      .delete(url, { ...this.httpOptions, observe: 'response' })
      .pipe(
        map((r: HttpResponse<unknown>) => r.status === 204),
        tap((ok) => this.log(ok ? `Deleted user id=${id}` : `Delete failed id=${id}`)),
        catchError(this.handleError<boolean>('deleteUser', false))
      );
  }

  httpOptions = {
    headers: new HttpHeaders({ 'Content-Type': 'application/json' })
  };

  private handleError<T>(operation = 'operation', result?: T) {
    return (error: unknown): Observable<T> => {
      console.error(error);
      let msg = error instanceof Error ? error.message : String(error);
      if (error instanceof HttpErrorResponse) {
        const apiErr = error.error?.error;
        const apiMsg =
          apiErr && typeof apiErr.message === 'string' ? apiErr.message : '';
        const apiCode = apiErr && typeof apiErr.code === 'string' ? apiErr.code : '';
        msg = `${error.status} ${error.statusText}`.trim();
        if (apiCode || apiMsg) {
          msg = `${msg} ${apiCode} ${apiMsg}`.trim();
        }
      }
      this.log(`${operation} failed: ${msg}`);
      return of(result as T);
    };
  }

  private log(message: string) {
    this.messageService.add(`UserService: ${message}`);
  }
}
