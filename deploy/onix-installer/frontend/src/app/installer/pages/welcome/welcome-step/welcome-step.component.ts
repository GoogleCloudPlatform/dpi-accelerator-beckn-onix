import { Component, Input, Output, EventEmitter } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { FormGroup, ReactiveFormsModule } from '@angular/forms';

@Component({
  selector: 'app-welcome-step',
  standalone: true,
  imports: [
    CommonModule, 
    MatButtonModule, 
    MatIconModule, 
    ReactiveFormsModule
  ],
  templateUrl: './welcome-step.component.html',
  styleUrls: ['./welcome-step.component.css']
})
export class WelcomeStepComponent {
  @Input() formGroup!: FormGroup;
  // Output event to notify the parent dialog to advance the step
  @Output() acknowledged = new EventEmitter<void>(); 

  constructor() {}

  acknowledgeAndProceed(): void {
    // 1. Mark the required form control as true (validating the step)
    this.formGroup.get('isAcknowledged')?.setValue(true);
    
    // 2. Emit event to tell parent (InstallerDialogComponent) to advance
    this.acknowledged.emit();
  }
}