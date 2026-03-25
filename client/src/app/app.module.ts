import { APP_INITIALIZER, NgModule } from '@angular/core';
import { BrowserModule } from '@angular/platform-browser';
import { FormsModule } from '@angular/forms';

import { AppRoutingModule } from './app-routing.module';
import { AppComponent } from './app.component';
import { UsersComponent } from './users/users.component';
import { UserDetailComponent } from './user-detail/user-detail.component';
import { MessagesComponent } from './messages/messages.component';
import { DashboardComponent } from './dashboard/dashboard.component';
import { HTTP_INTERCEPTORS, HttpClientModule } from '@angular/common/http';
import { UserSearchComponent } from './user-search/user-search.component';
import { LoginComponent } from './login/login.component';
import { RegisterComponent } from './register/register.component';
import { PlayersComponent } from './players/players.component';
import { CharactersComponent } from './characters/characters.component';
import { DataopsComponent } from './dataops/dataops.component';
import { SecurityComponent } from './security/security.component';
import { AuditComponent } from './audit/audit.component';
import { AuthService } from './auth.service';
import { AuthInterceptor } from './auth.interceptor';
import { CredentialsInterceptor } from './credentials.interceptor';
import { CsrfInterceptor } from './csrf.interceptor';
import { UnauthorizedInterceptor } from './unauthorized.interceptor';
import { firstValueFrom } from 'rxjs';

export function authAppInit(auth: AuthService) {
  return () => firstValueFrom(auth.initApp());
}

@NgModule({
  declarations: [
    AppComponent,
    UsersComponent,
    UserDetailComponent,
    MessagesComponent,
    DashboardComponent,
    UserSearchComponent,
    LoginComponent,
    RegisterComponent,
    PlayersComponent,
    CharactersComponent,
    DataopsComponent,
    SecurityComponent,
    AuditComponent
  ],
  imports: [BrowserModule, AppRoutingModule, FormsModule, HttpClientModule],
  providers: [
    {
      provide: APP_INITIALIZER,
      useFactory: authAppInit,
      deps: [AuthService],
      multi: true
    },
    { provide: HTTP_INTERCEPTORS, useClass: UnauthorizedInterceptor, multi: true },
    { provide: HTTP_INTERCEPTORS, useClass: CredentialsInterceptor, multi: true },
    { provide: HTTP_INTERCEPTORS, useClass: AuthInterceptor, multi: true },
    { provide: HTTP_INTERCEPTORS, useClass: CsrfInterceptor, multi: true }
  ],
  bootstrap: [AppComponent]
})
export class AppModule {}
