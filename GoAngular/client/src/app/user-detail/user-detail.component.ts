import { Component, OnInit, Input } from '@angular/core';
import {User} from '../user';
import { Router, ParamMap, ActivatedRoute} from '@angular/router';
import { Location } from '@angular/common';
import { UserService } from '../user.service';

@Component({
  selector: 'app-user-detail',
  templateUrl: './user-detail.component.html',
  styleUrls: ['./user-detail.component.css']
})
export class UserDetailComponent implements OnInit {
  user!: User;
  id!: Number;

  constructor(
    private route: ActivatedRoute,
    private userService: UserService,
    private location: Location
  ) { }

  ngOnInit() {
    this.id= this.id;
    this.getUser();
  }

  getUser(): void {
   this.id = Number(this.route.paramMap.subscribe(params => {
      const id = params.get('id');
      if (id != null) {
        this.id = +id;
      }    
    this.userService.getUser(this.id).subscribe(user => this.user = user);
  }))
  }

  save(): void {
    if (this.user) {
      this.userService.updateUser(this.user)
      .subscribe(() => this.goBack());
    }
  }

  goBack(): void {
    this.location.back();
  }
}
