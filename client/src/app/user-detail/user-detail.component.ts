import { Component, OnInit } from '@angular/core';
import {User} from '../user';
import { ActivatedRoute} from '@angular/router';
import { Location } from '@angular/common';
import { UserService } from '../user.service';
import { MessageService } from '../message.service';

@Component({
  selector: 'app-user-detail',
  templateUrl: './user-detail.component.html',
  styleUrls: ['./user-detail.component.css']
})
export class UserDetailComponent implements OnInit {
  user!: User;
  id!: number;
  saveError = '';

  constructor(
    private route: ActivatedRoute,
    private userService: UserService,
    private location: Location,
    private messageService: MessageService
  ) { }

  ngOnInit(): void {
    this.getUser();
  }

  getUser(): void {
    const id = this.route.snapshot.paramMap.get('id');
    if (!id) {
      return;
    }
    this.id = +id;
    this.userService.getUser(this.id).subscribe(user => this.user = user);
  }

  save(): void {
    if (this.user) {
      this.saveError = '';
      this.userService.updateUser(this.user)
      .subscribe((updated) => {
        if (!updated) {
          this.saveError = 'Save failed. Check messages for details.';
          return;
        }
        this.messageService.add(`Saved user id=${updated.id}`);
        this.goBack();
      });
    }
  }

  goBack(): void {
    this.location.back();
  }
}
