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

import { Injectable } from '@angular/core';
import { Observable, Subject, BehaviorSubject } from 'rxjs';
import { webSocket, WebSocketSubject } from 'rxjs/webSocket';
import { shareReplay } from 'rxjs/operators';

@Injectable({
  providedIn: 'root',
})
export class WebSocketService {
  private socket$: WebSocketSubject<any> | null = null;
  private connectionStatusSubject = new BehaviorSubject<boolean>(false);
  public connectionStatus$: Observable<boolean> = this.connectionStatusSubject.asObservable();

  constructor() {}

  /**
   * Connects to a WebSocket URL and returns an Observable for incoming messages.
   * If a connection already exists to the same URL, it returns the existing observable.
   * @param url The WebSocket URL to connect to.
   * @returns An Observable of messages received from the WebSocket.
   */
  connect(url: string): Observable<any> {
    if (!this.socket$ || this.socket$.closed) {
      this.socket$ = webSocket({
        url: url,
        openObserver: {
          next: () => {
            console.log('WebSocket connected:', url);
            this.connectionStatusSubject.next(true);
          },
        },
        closeObserver: {
          next: () => {
            console.log('WebSocket disconnected:', url);
            this.connectionStatusSubject.next(false);
          },
        },
      });
      return this.socket$.pipe(shareReplay(1));
    }
    return this.socket$;
  }

  /**
   * Sends a message over the active WebSocket connection.
   * @param message The message to send.
   */
  sendMessage(message: any): void {
    if (this.socket$ && !this.socket$.closed) {
      this.socket$.next(message);
    } else {
      console.warn('WebSocket is not connected. Message not sent:', message);
    }
  }

  closeConnection(): void {
    if (this.socket$) {
      this.socket$.complete();
      this.socket$ = null;
      this.connectionStatusSubject.next(false);
    }
  }
}