import { Injectable } from '@angular/core';
import { User } from './user';
import { Observable, of } from 'rxjs';
import { catchError, map, tap } from 'rxjs/operators';
import { MessageService } from './message.service';
import { HttpClient, HttpHeaders } from '@angular/common/http';

@Injectable({
  providedIn: 'root'
})
export class UserService {

  // URL to web api
  private usersUrl = 'api/users';

  constructor(
    private http: HttpClient,
    private messageService: MessageService
  ) { }

  //GET all users
  getUsers(): Observable<User[]> {
    return this.http.get<User[]>(this.usersUrl)
    .pipe(
      tap(_ => this.log('Fetched users')),
      catchError(this.handleError<User[]>('getUsers', []))
    );
  }

  //GET user by id
  getUser<Data>(id: Number): Observable<User> {
    const url = `${this.usersUrl}/?id=${id}`;
    return this.http.get<User[]>(url).pipe(
      map(users => users[0]),
      tap(u => {
        const outcome = u ? 'Fetched' : 'Did not find';
        this.log(`${outcome} user id=${id}`);
      }),
      catchError(this.handleError<User>(`getUser id=${id}`))
    );
  }

  //GET user by searched name
  searchUsers(term: string): Observable<User[]> {
    if (!term.trim()) {
      return of([]);
    }
    return this.http.get<User[]>(`${this.usersUrl}/?name=${term}`)
    .pipe(
      tap(x => x.length ?
        this.log(`Found users matching "${term}"`):
        this.log(`No users matching "${term}"`)),
      catchError(this.handleError<User[]>(`searchUsers`, []))
    );
  }

  // POST new user to the server
  addUser(user: User): Observable<User> {
    return this.http.post<User>(this.usersUrl, user, this.httpOptions).pipe(
      tap((newuser: User) => this.log(`Added user w/ id=${newuser.id}`)),
      catchError(this.handleError<User>('addUser'))
    );
  }

  //PUT saves updated details
  updateUser(user: User): Observable<any> {
    return this.http.put(this.usersUrl, user, this.httpOptions)
    .pipe(
      tap( _ => this.log(`Updated user id=${user.id}`)),
      catchError(this.handleError<any>('updateUser'))
    );
  }

  //DELETE user by id
  deleteUser(id: number): Observable<User> {
    const url = `${this.usersUrl}/${id}`;
    return this.http.delete<User>(url, this.httpOptions)
    .pipe(
      tap(_ => this.log(`Deleted user id=${id}`)),
      catchError(this.handleError<User>('deleteUser'))
    );
  }

  httpOptions = {
    headers: new HttpHeaders({ 'Content-Type': 'application/json' })
  };

  //Handle Http operation failure, let app continue
  //@param operation - name of the operation that failed
  //@param result - optional value to return as the observable result
  private handleError<T>(operation = 'operation', result?: T) {
    return (error: any): Observable<T> => {
      console.error(error); // log to console
      this.log(`${operation} failed: ${error.message}`);
      return of(result as T);
    };
  }

  private log(message: string) {
    this.messageService.add(`UserService: ${message}`);
  }
}