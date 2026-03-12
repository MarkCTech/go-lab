import { Component, OnInit } from '@angular/core';
import { Observable, Subject } from 'rxjs';
import { debounceTime, distinctUntilChanged, switchMap } from 'rxjs/operators';
import { User } from '../user';
import { UserService } from '../user.service';

@Component({
  selector: 'app-user-search',
  templateUrl: './user-search.component.html',
  styleUrls: ['./user-search.component.css']
})
export class UserSearchComponent implements OnInit {
  users$!: Observable<User[]>;
  private searchTerms = new Subject<string>();

  public show:boolean = false;
  public buttonName:any = 'Show';

  constructor(private userService: UserService) { }

  ngOnInit(): void {
    this.makeSearch();
  };

  search(term: string): void {
    this.searchTerms.next(term);
  }

  makeSearch(): void {
    this.users$ = this.searchTerms.pipe(
      // Wait ms on keystroke before considering
      debounceTime(300),
      // Ignore same term resubmission
      distinctUntilChanged(),
      // Observable change when term changes
      switchMap((term: string) => this.userService.searchUsers(term)),
    );
  }

  searchShow(): void {
    // Shows search results on clicking Input-Box
    this.show = true;
  }

  searchToggle(): void {
   // reverse the value of property
    this.show = !this.show;
    if(this.show)
      this.buttonName = "Hide";
    else
      this.buttonName = "Show";
  };
}

