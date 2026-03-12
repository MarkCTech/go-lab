import { Injectable } from '@angular/core';
import { InMemoryDbService } from 'angular-in-memory-web-api';
import { User } from './user';

@Injectable({
  providedIn: 'root',
})
export class InMemoryDataService implements InMemoryDbService {
  createDb() {
      const users = [
      {id:1, name: 'Tom', pennies: 11},
      {id:2, name: 'Dick', pennies: 22},
      {id:3, name: 'Harry', pennies: 33},
      {id:4, name: 'Peter', pennies: 44},
      {id:5, name: 'Paul', pennies: 55}
    ];
    // const users: User[] = [];
    return {users};
  }

  // Overrides the genId method to ensure that a user always has an id.
  // If the users array is empty, returns the initial number (11).
  // if the users array is not empty, the method below returns the highest
  // user id + 1.
  genId(users: User[]): number {
    return users.length > 0 ? Math.max(...users.map(user => user.id)) + 1 : 1;
  }
}