/**
 * Copyright 2025 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { Component, OnDestroy, ChangeDetectorRef, OnInit, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Subject, EMPTY } from 'rxjs';
import { takeUntil, catchError, finalize } from 'rxjs/operators';
import { WebSocketService } from '../../../core/services/websocket.service';
import { MatListItem } from '@angular/material/list';
import { InstallerStateService } from '../../../core/services/installer-state.service';
import { Router } from '@angular/router';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatButtonModule } from '@angular/material/button';
import { HealthCheckItem } from '../../types/installer.types';


@Component({
  changeDetection: ChangeDetectionStrategy.Eager,selector: 'app-step-health-check',
  standalone: true,
  imports: [CommonModule, MatIconModule, MatProgressSpinnerModule,MatButtonModule],
  templateUrl: './step-health-check.html',
  styleUrls: ['./step-health-check.css']
})
export class StepHealthCheck implements OnInit, OnDestroy {
  healthCheckStatus: 'idle' | 'inProgress' | 'success' | 'failed' = 'idle';
  showSuccessModal = false;
  checkResults: HealthCheckItem[] = [];
  logMessages: string[] = [];

  private  servicesToTest: { [key: string]: string } = {
  };

  private unsubscribe$ = new Subject<void>();

  constructor(
    private webSocketService: WebSocketService,
    private cdr: ChangeDetectorRef,
    private installerStateService : InstallerStateService,
       private router: Router
  ) {}

  ngOnInit(): void {
    const currentState =this.installerStateService.getCurrentState();

    console.log(currentState);
    currentState.componentSubdomainPrefixes
    const subdomainMap: { [key: string]: string } = currentState.componentSubdomainPrefixes
  .filter(config => config.component)
  .reduce((accumulator, currentConfig) => {
    accumulator[currentConfig.component] = currentConfig.subdomainPrefix!;
    return accumulator;
  }, {} as { [key: string]: string });
     console.log(subdomainMap);
 //   this.servicesToTest=subdomainMap;
    if(currentState?.deploymentGoal?.bap||currentState?.deploymentGoal?.bpp){
      this.servicesToTest['Adapter']=subdomainMap['adapter'];
      this.servicesToTest['Subscriber']= subdomainMap['subscriber'];
    }
    if(currentState?.deploymentGoal?.registry||currentState?.deploymentGoal?.registry){
      this.servicesToTest['Registry']=subdomainMap['registry'];
      this.servicesToTest['Registry-Admin']=subdomainMap['registry_admin'];
    }
    if(currentState?.deploymentGoal?.gateway||currentState?.deploymentGoal?.gateway){
      this.servicesToTest['Gateway']=subdomainMap['gateway'];
      this.servicesToTest['Subscriber']=subdomainMap['subscriber'];
    }
    this.initializeChecksFromConfig();

  }

  ngOnDestroy(): void {
    this.unsubscribe$.next();
    this.unsubscribe$.complete();
    this.webSocketService.closeConnection();
  }

  private initializeChecksFromConfig(): void {
    this.checkResults = Object.keys(this.servicesToTest).map(name => ({
      name: name,
      url: this.servicesToTest[name],
      status: null
    }));
  }

  runHealthCheck(): void {
    if (this.healthCheckStatus === 'inProgress') {
      return;
    }

    this.healthCheckStatus = 'inProgress';
    this.showSuccessModal = false;
    this.logMessages = [];
    this.checkResults.forEach(item => item.status = 'pending');
    this.cdr.detectChanges();

    const payload = this.servicesToTest;
    const wsUrl = 'ws://localhost:8000/ws/healthCheck';

    this.webSocketService.connect(wsUrl)
      .pipe(
        takeUntil(this.unsubscribe$),
        catchError(error => {
          console.error('WebSocket connection failed:', error);
          this.setFinalStatus('failed');
          return EMPTY;
        }),
        finalize(() => {
          if (this.healthCheckStatus === 'inProgress') {
            console.warn('WebSocket disconnected before all checks were complete.');
            this.setFinalStatus('failed');
          }
        })
      )
      .subscribe(message => {
        this.handleWebSocketMessage(message);
        this.cdr.detectChanges();
      });
    this.webSocketService.sendMessage(payload);
  }

  /**
   * Processes messages from the WebSocket to update the UI state.
   * @param message The JSON message object from the server.
   */
  private handleWebSocketMessage(message: any): void {
  if (this.healthCheckStatus !== 'inProgress') {
    return;
  }

  console.log('Received from WS:', message);

  if (message.service) {
    const item = this.checkResults.find(r => r.name === message.service);
    if (item) {
      if (message.type === 'success') {
        item.status = 'success';
      }

      else if (message.type === 'error' && message.service === item.name) {
          item.status = 'failed';
      }
      console.log(`Updated status for ${item.name}: ${item.status}`);

    }
  }

  this.logMessages.push(message.message);

  if (message.action === 'all_services_healthy') {
    this.setFinalStatus('success');
    this.webSocketService.closeConnection();
  } else if (message.action === 'health_check_timeout') {
    this.setFinalStatus('failed');
    this.logMessages.push(`Error: ${message.message}`);
    this.webSocketService.closeConnection();
  }

  if (message.type === 'error' && !message.action && !message.service) {
    this.setFinalStatus('failed');
    this.logMessages.push(`Error: ${message.message}`);
    this.webSocketService.closeConnection();
  }

  this.cdr.detectChanges();
}

   onNext(): void {
      this.router.navigate(['installer', 'subscribe']);
  }

  /**
   * Sets the final state of the component (success or failed) and updates any remaining pending items.
   * @param status The final status to set.
   */
  private setFinalStatus(status: 'success' | 'failed' ): void {
    if (this.healthCheckStatus !== 'inProgress') return;

    this.healthCheckStatus = status;

    if (status === 'success') {
      this.showSuccessModal = true;
      this.checkResults.forEach(item => item.status = 'success');
    } else {
      this.checkResults.forEach(item => {
        if (item.status === 'pending' || item.status === null) {
          item.status = 'failed';
        }
      });
    }
    this.cdr.detectChanges();
  }

  retryHealthCheck(): void {
    this.runHealthCheck();
  }

  closeModal(): void {
    this.showSuccessModal = false;
    this.cdr.detectChanges();
  }
}