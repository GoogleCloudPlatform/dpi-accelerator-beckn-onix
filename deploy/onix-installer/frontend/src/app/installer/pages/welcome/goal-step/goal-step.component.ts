import { Component, Input, OnInit, OnDestroy } from '@angular/core'; // Removed Output, EventEmitter
import { CommonModule } from '@angular/common';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatRadioModule } from '@angular/material/radio';
import { MatCheckboxModule } from '@angular/material/checkbox';
import { MatFormFieldModule } from '@angular/material/form-field';
import { ReactiveFormsModule, FormGroup, FormControl } from '@angular/forms';
import { trigger, state, style, transition, animate } from '@angular/animations';
import { Subscription } from 'rxjs';

@Component({
  selector: 'app-goal-step',
  standalone: true,
  imports: [
    CommonModule,
    ReactiveFormsModule,
    MatButtonModule,
    MatIconModule,
    MatRadioModule,
    MatCheckboxModule,
    MatFormFieldModule
  ],
  templateUrl: './goal-step.component.html',
  styleUrls: ['./goal-step.component.css'],
  animations: [
    trigger('expandCollapse', [
      state('collapsed, void', style({ height: '0', paddingTop: '0', paddingBottom: '0', opacity: '0', overflow: 'hidden' })),
      state('expanded', style({ height: '*', paddingTop: '15px', paddingBottom: '15px', opacity: '1', overflow: 'hidden' })),
      transition('collapsed <=> expanded', animate('300ms ease-in-out'))
    ])
  ]
})
export class GoalStepComponent implements OnInit, OnDestroy {
  @Input() formGroup!: FormGroup;
  // REMOVED @Output() nextStep
  // REMOVED @Output() previousStep

  newNetworkKeys = ['registry', 'gateway', 'bap', 'bpp'];
  existingNetworkKeys = ['bap', 'bpp'];

  private subscriptions = new Subscription();

  get deploymentTypeControl(): FormControl {
    return this.formGroup.get('deploymentType') as FormControl;
  }

  get newNetworkComponentsGroup(): FormGroup {
    return this.formGroup.get('newNetworkComponents') as FormGroup;
  }

  get existingNetworkComponentsGroup(): FormGroup {
    return this.formGroup.get('existingNetworkComponents') as FormGroup;
  }

  constructor() {}

  ngOnInit(): void {
    this.subscriptions.add(
      this.deploymentTypeControl.valueChanges.subscribe(value => {
        // Re-enable the group to ensure its validity contributes to the parent formGroup
        if (value === 'new') {
            this.newNetworkComponentsGroup.enable({ emitEvent: false });
            this.existingNetworkComponentsGroup.disable({ emitEvent: false });
        } else if (value === 'join') {
            this.newNetworkComponentsGroup.disable({ emitEvent: false });
            this.existingNetworkComponentsGroup.enable({ emitEvent: false });
        }
        this.formGroup.updateValueAndValidity();
      })
    );

    this.setupAllToIndividualSync(this.newNetworkComponentsGroup, this.newNetworkKeys);
    this.setupAllToIndividualSync(this.existingNetworkComponentsGroup, this.existingNetworkKeys);
  }

  private setupAllToIndividualSync(group: FormGroup, keys: string[]): void {
    const allControl = group.get('all');
    if (allControl) {
      this.subscriptions.add(
        allControl.valueChanges.subscribe(isChecked => {
          const patchValue = keys.reduce((acc: { [key: string]: boolean }, key) => {
            acc[key] = isChecked;
            return acc;
          }, {});
          group.patchValue(patchValue, { emitEvent: false });
          this.onIndividualChange(group, keys); // Update validity
        })
      );
    }
  }

  onIndividualChange(group: FormGroup, keys: string[]): void {
    const allControl = group.get('all');
    if (allControl) {
      const allChecked = keys.every(key => group.get(key)?.value);
      if (allControl.value !== allChecked) {
        allControl.setValue(allChecked, { emitEvent: false });
      }
    }
    group.updateValueAndValidity();
    this.formGroup.updateValueAndValidity(); // Propagate validity up
  }

  // REMOVED isStepValid() - Parent handles this
  // REMOVED goToNextStep()
  // REMOVED goToPreviousStep()

  ngOnDestroy(): void {
    this.subscriptions.unsubscribe();
  }
}
