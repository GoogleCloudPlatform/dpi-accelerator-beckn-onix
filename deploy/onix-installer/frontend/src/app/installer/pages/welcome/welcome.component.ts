import { Component } from '@angular/core';
import { Router } from '@angular/router';
import { CommonModule } from '@angular/common';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatDialog, MatDialogModule } from '@angular/material/dialog';

import { InstallerDialogComponent } from './installer-dialog/installer-dialog.component';

@Component({
  selector: 'app-welcome',
  templateUrl: './welcome.component.html',
  styleUrls: ['./welcome.component.css'],
  standalone: true,
  imports: [CommonModule, MatButtonModule, MatIconModule]
})
export class WelcomeComponent {
  // Assume this is loaded from an environment variable
  selectedProjectId: string = 'your-gcp-project-id'; 

  constructor(
    private router: Router,
    private dialog: MatDialog
  ) { }

  goToCreateDeployment() {
    // this.router.navigate(['/installer/goal']);
    this.dialog.open(InstallerDialogComponent, {
      width: '90vw', // Full width for the large wizard
      height: '90vh', // Full height
      maxWidth: '1200px',
      maxHeight: '900px',
      disableClose: true, // User must use the X button inside the dialog
      panelClass: 'full-screen-dialog' // Custom class for styling the container
    });
  }
}
