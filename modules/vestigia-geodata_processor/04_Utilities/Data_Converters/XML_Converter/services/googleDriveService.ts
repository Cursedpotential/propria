
const SCOPES = 'https://www.googleapis.com/auth/drive.readonly';
const DISCOVERY_DOCS = ['https://www.googleapis.com/discovery/v1/apis/drive/v3/rest'];

export class GoogleDriveService {
  private tokenClient: any;
  private accessToken: string | null = null;
  private isInitialized = false;
  private clientId: string | null = null;

  constructor() {}

  initialize(clientId: string): Promise<void> {
    this.clientId = clientId;
    return new Promise((resolve) => {
      if (window.gapi && window.google) {
        this.loadGapi(resolve, clientId);
      } else {
        // Wait for scripts to load
        const interval = setInterval(() => {
          if (window.gapi && window.google) {
            clearInterval(interval);
            this.loadGapi(resolve, clientId);
          }
        }, 100);
      }
    });
  }

  private loadGapi(callback: () => void, clientId: string) {
    if (this.isInitialized && this.tokenClient) return callback();

    window.gapi.load('client:picker', async () => {
      await window.gapi.client.init({
        discoveryDocs: DISCOVERY_DOCS,
      });

      this.tokenClient = window.google.accounts.oauth2.initTokenClient({
        client_id: clientId,
        scope: SCOPES,
        callback: (response: any) => {
          if (response.error !== undefined) {
            throw response;
          }
          this.accessToken = response.access_token;
        },
      });
      
      this.isInitialized = true;
      callback();
    });
  }

  async handleAuthClick(clientId: string): Promise<string> {
    if (!this.isInitialized || !this.tokenClient) await this.initialize(clientId);

    return new Promise((resolve, reject) => {
      this.tokenClient.callback = (resp: any) => {
        if (resp.error) reject(resp);
        this.accessToken = resp.access_token;
        resolve(resp.access_token);
      };

      if (window.gapi.client.getToken() === null) {
        this.tokenClient.requestAccessToken({ prompt: 'consent' });
      } else {
        this.tokenClient.requestAccessToken({ prompt: '' });
      }
    });
  }

  createPicker(clientId: string): Promise<{ id: string; name: string; size: number; token: string } | null> {
    return new Promise(async (resolve, reject) => {
        if (!clientId) {
            reject("Client ID is missing. Check Settings.");
            return;
        }

        if (!this.accessToken) {
            try {
                await this.handleAuthClick(clientId);
            } catch(e) {
                reject("Auth failed");
                return;
            }
        }

        const view = new window.google.picker.DocsView(window.google.picker.ViewId.DOCS);
        view.setMimeTypes('text/xml'); // Filter for XML
        view.setIncludeFolders(true);

        const picker = new window.google.picker.PickerBuilder()
            .setDeveloperKey('') // Token is sufficient for picker usually, or specific api key
            .setAppId(clientId.split('-')[0])
            .setOAuthToken(this.accessToken!)
            .addView(view)
            .addView(new window.google.picker.DocsView()) // Allow all just in case
            .setCallback((data: any) => {
                if (data.action === window.google.picker.Action.PICKED) {
                    const file = data.docs[0];
                    resolve({
                        id: file.id,
                        name: file.name,
                        size: file.sizeBytes || 0,
                        token: this.accessToken!
                    });
                } else if (data.action === window.google.picker.Action.CANCEL) {
                    resolve(null);
                }
            })
            .build();
        
        picker.setVisible(true);
    });
  }

  async getFileStream(fileId: string, token: string): Promise<ReadableStreamDefaultReader<Uint8Array>> {
     const response = await fetch(`https://www.googleapis.com/drive/v3/files/${fileId}?alt=media`, {
         headers: {
             'Authorization': `Bearer ${token}`
         }
     });

     if (!response.ok) throw new Error(`Drive Fetch Error: ${response.statusText}`);
     if (!response.body) throw new Error("No body in response");
     
     return response.body.getReader();
  }
}
