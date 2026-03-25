import { Component, OnInit } from '@angular/core';
import { PlatformService } from '../platform.service';

@Component({
  selector: 'app-characters',
  templateUrl: './characters.component.html',
  styleUrls: ['./characters.component.css']
})
export class CharactersComponent implements OnInit {
  data: unknown = null;

  constructor(private platform: PlatformService) {}

  ngOnInit(): void {
    this.platform.getCharacters().subscribe((d) => (this.data = d));
  }
}
