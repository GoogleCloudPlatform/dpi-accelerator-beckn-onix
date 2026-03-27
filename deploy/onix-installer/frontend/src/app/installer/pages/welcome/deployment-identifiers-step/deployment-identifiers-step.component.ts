import { Component, Input, Output, EventEmitter, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatSelectModule } from '@angular/material/select';
import { ReactiveFormsModule, FormGroup, FormArray, FormBuilder, Validators } from '@angular/forms';

@Component({
  selector: 'app-deployment-identifiers-step',
  standalone: true,
  imports: [
    CommonModule, 
    ReactiveFormsModule,
    MatButtonModule, 
    MatIconModule,
    MatFormFieldModule,
    MatInputModule,
    MatSelectModule
  ],
  templateUrl: './deployment-identifiers-step.component.html',
  styleUrls: ['./deployment-identifiers-step.component.css']
})
export class DeploymentIdentifiersStepComponent implements OnInit {
  @Input() formGroup!: FormGroup;
  @Output() nextStep = new EventEmitter<void>(); 
  @Output() previousStep = new EventEmitter<void>();

  regions: string[] = ['us-central1 (Iowa)', 'us-east1 (South Carolina)', 'europe-west1 (Belgium)'];

  constructor(private fb: FormBuilder) {}

  ngOnInit(): void {}

  get projectTags(): FormArray {
    return this.formGroup.get('projectTags') as FormArray;
  }

  addTag(): void {
    this.projectTags.push(this.fb.control('', Validators.required));
  }

  removeTag(index: number): void {
    if (this.projectTags.length > 1) {
      this.projectTags.removeAt(index);
    }
  }

  // Logic: Only show "Add more" if the last field is not empty
  canAddMoreTags(): boolean {
    const controls = this.projectTags.controls;
    if (controls.length === 0) return true;
    const lastValue = controls[controls.length - 1].value;
    return lastValue && lastValue.trim().length > 0;
  }

  goToNextStep(): void {
    if (this.formGroup.valid) {
        const tagsArray = this.projectTags.value;
        const tagsJson: { [key: string]: string } = {};
        
        tagsArray.forEach((tag: string, index: number) => {
            if (tag.includes(',')) {
              const [key, value] = tag.split(',').map(s => s.trim());
              tagsJson[key] = value;
            } else {
              tagsJson[`tag_${index}`] = tag;
            }
        });
        console.log('Final Tags JSON:', tagsJson);
        this.nextStep.emit();
    }
  }

  goToPreviousStep(): void {
    this.previousStep.emit();
  }
}