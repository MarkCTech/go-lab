import { Component, OnInit } from '@angular/core';
import { PlatformService } from '../platform.service';

@Component({
  selector: 'app-security',
  templateUrl: './security.component.html',
  styleUrls: ['./security.component.css']
})
export class SecurityComponent implements OnInit {
  me: unknown = null;
  ackReason = '';
  ackMessage = '';
  ackResult: unknown = null;

  constructor(private platform: PlatformService) {}

  ngOnInit(): void {
    this.platform.getSecurityMe().subscribe((d) => (this.me = d));
  }

  submitAck(): void {
    this.ackResult = null;
    const r = this.ackReason.trim();
    if (r.length < 10) {
      return;
    }
    this.platform.postSupportAck(r, this.ackMessage).subscribe((d) => (this.ackResult = d));
  }
}
