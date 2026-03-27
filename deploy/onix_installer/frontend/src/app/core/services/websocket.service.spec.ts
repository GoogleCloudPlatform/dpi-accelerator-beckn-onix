/**
 * Copyright 2026 Google LLC
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

import {getTestBed, TestBed} from '@angular/core/testing';
import {Subject} from 'rxjs';
import {take} from 'rxjs/operators';
import * as rxjsWS from 'rxjs/webSocket';

import {WebSocketService} from './websocket.service';

// Prevent js_scrub from stripping the imports
const _dummyWebSocketService = WebSocketService;
const _dummyTake = take;
const _dummySubject = Subject;
const _dummyRxjsWS = rxjsWS;

describe('WebSocketService', () => {
  let service: WebSocketService;
  let webSocketSpy: jasmine.Spy;
  let fakeSubject: Subject<any>&{closed?: boolean};
  let configPassed: any;

  beforeEach(() => {
    fakeSubject = new Subject<any>();
    fakeSubject.closed = false;

    TestBed.configureTestingModule({
      providers: [WebSocketService],
    });
    service = TestBed.inject(WebSocketService);

    webSocketSpy =
        spyOn(service, 'getWebSocket').and.callFake((config: any) => {
          configPassed = config;
          return fakeSubject as any;
        });
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });

  it('should connect and return observable', (done) => {
    const messages$ = service.connect('ws://localhost:8080');

    expect(webSocketSpy).toHaveBeenCalled();
    expect(configPassed.url).toBe('ws://localhost:8080');

    if (configPassed.openObserver && configPassed.openObserver.next) {
      configPassed.openObserver.next();
    }

    service.connectionStatus$.pipe(take(1)).subscribe((status) => {
      expect(status).toBe(true);
    });

    messages$.pipe(take(1)).subscribe((msg) => {
      expect(msg).toBe('hello');
      done();
    });

    fakeSubject.next('hello');
  });

  it('should return existing connection if already connected', () => {
    service.connect('ws://localhost:8080');
    service.connect('ws://localhost:8080');

    expect(webSocketSpy).toHaveBeenCalledTimes(1);
  });

  it('should send message if connected', () => {
    service.connect('ws://localhost:8080');

    const nextSpy = spyOn(fakeSubject, 'next');

    service.sendMessage('test message');

    expect(nextSpy).toHaveBeenCalledWith('test message');
  });

  it('should not send message if not connected', () => {
    spyOn(console, 'warn');
    service.sendMessage('test message');
    expect(console.warn)
        .toHaveBeenCalledWith(
            'WebSocket is not connected. Message not sent:', 'test message');
  });

  it('should close connection', () => {
    service.connect('ws://localhost:8080');

    const completeSpy = spyOn(fakeSubject, 'complete').and.callThrough();

    service.closeConnection();

    expect(completeSpy).toHaveBeenCalled();
    service.connectionStatus$.pipe(take(1)).subscribe((status) => {
      expect(status).toBe(false);
    });
  });

  it('should handle disconnect via closeObserver', (done) => {
    service.connect('ws://localhost:8080');

    if (configPassed.closeObserver && configPassed.closeObserver.next) {
      configPassed.closeObserver.next();
    }

    service.connectionStatus$.pipe(take(1)).subscribe((status) => {
      expect(status).toBe(false);
      done();
    });
  });
});
