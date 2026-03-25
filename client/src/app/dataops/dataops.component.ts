import { Component, OnInit } from '@angular/core';
import { PlatformService } from '../platform.service';

@Component({
  selector: 'app-dataops',
  templateUrl: './dataops.component.html',
  styleUrls: ['./dataops.component.css']
})
export class DataopsComponent implements OnInit {
  data: unknown = null;

  constructor(private platform: PlatformService) {}

  ngOnInit(): void {
    this.platform.getBackupsStatus().subscribe((d) => (this.data = d));
  }
}
