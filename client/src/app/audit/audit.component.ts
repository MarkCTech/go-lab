import { Component, OnInit } from '@angular/core';
import { PlatformService } from '../platform.service';

@Component({
  selector: 'app-audit',
  templateUrl: './audit.component.html',
  styleUrls: ['./audit.component.css']
})
export class AuditComponent implements OnInit {
  data: unknown = null;

  constructor(private platform: PlatformService) {}

  ngOnInit(): void {
    this.platform.getAdminAuditEvents().subscribe((d) => (this.data = d));
  }
}
