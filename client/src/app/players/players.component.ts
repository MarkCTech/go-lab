import { Component, OnInit } from '@angular/core';
import { PlatformService } from '../platform.service';

@Component({
  selector: 'app-players',
  templateUrl: './players.component.html',
  styleUrls: ['./players.component.css']
})
export class PlayersComponent implements OnInit {
  data: unknown = null;

  constructor(private platform: PlatformService) {}

  ngOnInit(): void {
    this.platform.getPlayers().subscribe((d) => (this.data = d));
  }
}
