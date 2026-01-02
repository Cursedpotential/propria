import { ParsedMessage, ConversionConfig, Base64Option, FileSystemDirectoryHandle, FileSystemWritableFileStream, ProcessingMetadata, SplitMethod, StreamSource, DeduplicationStrategy } from '../types';
import { Sha256 } from './sha256';
import { UuidV7 } from './uuid';
import { DateUtils } from './dateUtils';
import { SupabaseService } from './supabaseService';
import { SqliteService } from './sqliteService';
import { GoogleDriveService } from './googleDriveService';
import { BrowserNLPService } from './browserNlpService';

const CHUNK_SIZE = 1024 * 1024; 
const SUPABASE_BATCH_SIZE = 500;

export class XmlStreamProcessor {
  private source: StreamSource;
  private config: ConversionConfig;
  private abortController: AbortController;
  private hasher: Sha256 | null = null;
  private metadata: ProcessingMetadata;
  private dateUtils: DateUtils;
  private supabase: SupabaseService | null = null;
  private sqlite: SqliteService | null = null;
  private driveService: GoogleDriveService | null = null;
  private nlpService: BrowserNLPService | null = null;

  // Forensic tracking
  private documentId: string | null = null;
  private conversationCache: Map<string, string> = new Map(); // phone+platform -> conversationId

  // Buffers
  private msgBuffer: any[] = [];
  private entityBuffer: Map<string, any> = new Map();
  private attachmentBuffer: any[] = [];

  // Neo4j Buffers - Dynamic
  // Map<Label, Map<Id, Props>>
  private neo4jNodes: Map<string, Map<string, any>> = new Map(); 

  private currentSplitIndex = 1;
  private currentSplitSize = 0;
  private currentSplitRows = 0;
  private currentSplitKey: string = ''; // Used for date tracking (e.g., '2023-10')

  constructor(source: StreamSource, config: ConversionConfig) {
    this.source = source;
    this.config = config;
    this.abortController = new AbortController();
    this.dateUtils = new DateUtils();

    if (config.calculateHash) {
      this.hasher = new Sha256();
    }
    
    if (source.type === 'DRIVE') {
        this.driveService = new GoogleDriveService();
    }
    
    const needsSupabase = config.streamToSupabase || config.base64Option === Base64Option.UPLOAD_TO_STORAGE;
    if (needsSupabase && config.supabaseUrl && config.supabaseKey) {
        this.supabase = new SupabaseService(
            config.supabaseUrl, 
            config.supabaseKey, 
            config.tableName,
            config.entityTable,
            config.attachmentTable
        );
    }
    
    if (config.exportSqliteFile) {
        this.sqlite = new SqliteService();
    }

    // Initialize NLP service if Supabase enabled (for forensic analysis)
    if (needsSupabase) {
        this.nlpService = new BrowserNLPService();
        this.nlpService.setEnabled(true);
    }

    this.metadata = {
      originalFileName: source.type === 'FILE' ? source.file.name : source.name,
      fileSize: source.type === 'FILE' ? source.file.size : source.size,
      lastModified: Date.now(),
      startTime: new Date().toISOString(),
      recordCount: 0,
      validationErrors: 0,
      participants: new Set()
    };
  }

  abort() {
    this.abortController.abort();
  }

  private async getReader(): Promise<ReadableStreamDefaultReader<Uint8Array>> {
      if (this.source.type === 'FILE') {
          return this.source.file.stream().getReader();
      } else {
          // Drive Stream
          if (!this.driveService) throw new Error("Drive Service not initialized");
          return await this.driveService.getFileStream(this.source.url, this.source.token);
      }
  }

  async scanForColumns(limit: number = 50): Promise<{ preview: ParsedMessage[], allKeys: Set<string>, detectedTag: string, confidence: number }> {
    const results: ParsedMessage[] = [];
    const allKeys = new Set<string>();
    let errorCount = 0;
    
    // Core Columns
    if (this.config.generateUuid) allKeys.add('uuid_v7');
    if (this.config.calculateHash) allKeys.add('record_hash'); 
    allKeys.add('source_id'); 
    allKeys.add('date_iso');
    if (this.config.humanReadableTime) allKeys.add('date_human_est');
    allKeys.add('image_files');
    if (this.config.base64Option === Base64Option.INLINE) allKeys.add('base64_content');

    const reader = await this.getReader();
    const decoder = new TextDecoder();
    let buffer = '';
    let itemTag = this.config.itemTag;
    
    try {
      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        buffer += decoder.decode(value, { stream: true });
        
        // Auto-detect tag if not set
        if (!itemTag) {
           if (buffer.includes('<mms')) itemTag = 'mms';
           else if (buffer.includes('<sms')) itemTag = 'sms';
           else if (buffer.includes('<call')) itemTag = 'call';
        }

        if (itemTag) {
            // Regex to match call logs which might be self-closing <call ... />
            const tagRegex = new RegExp(`<${itemTag}[^>]*>.*?</${itemTag}>|<${itemTag}[^>]*/>`, 'gs');
            let match;
            while ((match = tagRegex.exec(buffer)) !== null) {
                const rawXml = match[0];
                const parsed = this.processItem(rawXml, itemTag);
                if (parsed) {
                    // Validation Check (relaxed for calls which might not have body)
                    if (!parsed.date_iso && !parsed.date) errorCount++;
                    
                    Object.keys(parsed).forEach(k => {
                        if (k !== '_parts' && k !== '_addrs') allKeys.add(k);
                    });
                    if (results.length < limit) results.push(parsed);
                }
                if (results.length >= limit * 2) { 
                    await reader.cancel(); 
                    break; 
                }
            }
            if (results.length >= limit) break;
            
            // Buffer management
            const lastTagIndex = buffer.lastIndexOf(`<${itemTag}`);
            if (lastTagIndex > -1) {
                buffer = buffer.substring(lastTagIndex);
            } else if (buffer.length > CHUNK_SIZE * 2) {
                 // Safe truncation for scanning
                 const lastTagStart = buffer.lastIndexOf(`<${itemTag}`);
                 if (lastTagStart === -1) {
                     buffer = buffer.substring(buffer.length - CHUNK_SIZE);
                 } else {
                     buffer = buffer.substring(lastTagStart);
                 }
            }
        } else if (buffer.length > CHUNK_SIZE * 2) {
            // Drop start if we haven't found a tag yet
            buffer = buffer.substring(buffer.length - CHUNK_SIZE);
        }
      }
    } catch (e) {
        console.error("Scan error", e);
    } finally {
      // Reader lock release handled by break/cancel
    }
    
    const confidence = results.length > 0 ? Math.max(0, 100 - (errorCount / results.length * 100)) : 0;
    
    return { preview: results.slice(0, limit), allKeys, detectedTag: itemTag || 'sms', confidence };
  }

  async convert(outputDir: FileSystemDirectoryHandle | null, onProgress: (processed: number) => void): Promise<void> {
    const reader = await this.getReader();
    const decoder = new TextDecoder();
    let buffer = '';

    const baseName = this.metadata.originalFileName.replace(/\.xml$/i, '');
    let csvWritable: FileSystemWritableFileStream | null = null;
    let sqlWritable: FileSystemWritableFileStream | null = null;

    // Neo4j Handles
    let neo4jRelsWritable: FileSystemWritableFileStream | null = null;

    // CREATE DOCUMENT RECORD (Forensic Chain of Custody)
    if (this.supabase && this.config.streamToSupabase) {
        const documentUuid = UuidV7.generate();
        const fileHash = await this.calculateFileHash();

        this.documentId = await this.supabase.createDocument({
            id: documentUuid,
            filename: this.metadata.originalFileName,
            file_hash: fileHash || 'PENDING',
            acquired_by: this.config.sourceLabel || 'Device Owner',
            acquired_date: new Date().toISOString(),
            acquisition_method: 'XML Export - Browser Ingestion',
            source_label: this.config.sourceLabel,
            file_size_bytes: this.metadata.fileSize,
            notes: `Processing started: ${new Date().toISOString()}`
        });

        console.log(`📄 Document created: ${this.documentId}`);
    }

    if (this.supabase && this.config.base64Option === Base64Option.UPLOAD_TO_STORAGE && this.config.storageBucket) {
        await this.supabase.createBucket(this.config.storageBucket);
    }

    if (this.sqlite) {
        await this.sqlite.init(this.config.columns, this.config.tableName);
    }
    
    if (this.config.exportNeo4j && this.config.exportLocalFile && outputDir) {
        const relsFile = await outputDir.getFileHandle(`${baseName}_relationships.csv`, { create: true });
        neo4jRelsWritable = await relsFile.createWritable();
        await neo4jRelsWritable.write(`:START_ID,:END_ID,:TYPE,date,uuid\n`);
        
        // Initialize dynamic node stores
        if (this.config.graphConfig) {
            this.config.graphConfig.nodes.forEach(n => {
                this.neo4jNodes.set(n.label, new Map());
            });
        }
        
        // Ensure "Person" label exists if using default
        if (!this.neo4jNodes.has("Person")) this.neo4jNodes.set("Person", new Map());
        this.neo4jNodes.get("Person")?.set("ME", { name: "Device Owner" });
    }

    const openNewSplit = async (discriminator: string = '') => {
        if (!this.config.exportLocalFile || !outputDir) return;

        if (csvWritable) await csvWritable.close();
        if (sqlWritable) await sqlWritable.close();

        // Dynamic File Naming Logic
        const seq = String(this.currentSplitIndex).padStart(3, '0');
        const sourceLabel = this.config.sourceLabel || baseName;
        const dateStr = discriminator || 'All';
        
        let filename = this.config.filenamePattern || "{Source}_{Date}_{Seq}";
        filename = filename
            .replace('{Source}', sourceLabel)
            .replace('{Date}', dateStr)
            .replace('{Seq}', seq);
        
        filename = filename.replace(/[^a-zA-Z0-9_\-]/g, '_');

        const csvFile = await outputDir.getFileHandle(`${filename}.csv`, { create: true });
        csvWritable = await csvFile.createWritable();
        await csvWritable.write(this.config.columns.join(',') + '\n');
        
        if (this.config.exportLocalFile && this.config.localExportFormat === 'POSTGRES') { 
            const sqlFile = await outputDir.getFileHandle(`${filename}.sql`, { create: true });
            sqlWritable = await sqlFile.createWritable();
            await this.writeSqlHeader(sqlWritable, filename);
        }
        this.currentSplitSize = 0;
        this.currentSplitRows = 0;
        this.currentSplitIndex++;
    };

    if (this.config.exportLocalFile && this.config.splitMethod !== SplitMethod.DATE) {
        await openNewSplit('Full');
    }

    let partsCsvWritable: FileSystemWritableFileStream | null = null;
    if (this.config.exportLocalFile && this.config.base64Option === Base64Option.SEPARATE_FILE && outputDir) {
         const partsFile = await outputDir.getFileHandle(`${baseName}_parts.csv`, { create: true });
         partsCsvWritable = await partsFile.createWritable();
         await partsCsvWritable.write(`parent_uuid,parent_date,part_seq,content_type,name,text_content,filename\n`);
    }

    let imgDirHandle: FileSystemDirectoryHandle | null = null;
    if (this.config.exportLocalFile && this.config.base64Option === Base64Option.EXPORT_IMAGES && outputDir) {
        imgDirHandle = await outputDir.getDirectoryHandle('images', { create: true });
    }

    let itemTag = this.config.itemTag || 'sms';
    const itemRegex = new RegExp(`<${itemTag}(\\s+[^>]*)?(\\/>|>(.*?)<\\/${itemTag}>)`, 'gs');

    try {
      while (true) {
        if (this.abortController.signal.aborted) throw new Error("Aborted");
        const { done, value } = await reader.read();
        if (done) break;

        // if (this.hasher) this.hasher.update(value); // Removed synchronous hasher
        buffer += decoder.decode(value, { stream: true });
        
        let match;
        let lastLastIndex = 0;

        while ((match = itemRegex.exec(buffer)) !== null) {
          const rawXml = match[0];
          lastLastIndex = itemRegex.lastIndex;
          const parsed = this.processItem(rawXml, itemTag);

          if (parsed) {
            // Validation
            if (!parsed.date_iso && !parsed.date) {
                this.metadata.validationErrors++;
                // Optionally skip invalid records? For now we keep them but log error count.
            }

            this.updateMetadata(parsed);
            parsed['source_id'] = this.config.sourceLabel || this.metadata.originalFileName;

            // --- Robust Deduplication Logic ---
            if (this.config.calculateHash) {
                const norm = (val: any) => (val === undefined || val === null) ? '' : String(val).trim();
                let contentSignature = '';

                switch (this.config.deduplicationStrategy) {
                    case DeduplicationStrategy.ID_ONLY:
                        // Fastest, trusts source IDs
                        contentSignature = `${norm(parsed.guid || parsed.uuid || parsed._id)}`;
                        break;
                    case DeduplicationStrategy.FUZZY_BODY:
                         // Ignore whitespace and case in body, use rough date (day only)
                         const roughDate = norm(parsed.date_iso).substring(0, 10);
                         const fuzzyBody = norm(parsed.body).toLowerCase().replace(/\s+/g, ' ');
                         contentSignature = `${roughDate}|${norm(parsed.address)}|${fuzzyBody}`;
                         break;
                    case DeduplicationStrategy.STRICT_HASH:
                    default:
                        // Exact match of all key fields
                        contentSignature = `${norm(parsed.date_iso)}|${norm(parsed.address)}|${norm(parsed.body)}|${norm(parsed.type)}`;
                        break;
                }
                
                const encoder = new TextEncoder();
                const data = encoder.encode(contentSignature);
                const hashBuffer = await crypto.subtle.digest('SHA-256', data);
                const hashArray = Array.from(new Uint8Array(hashBuffer));
                parsed['record_hash'] = hashArray.map(b => b.toString(16).padStart(2, '0')).join('');
            }

            // NLP ANALYSIS (Micro-Level Surface Tagging)
            if (this.nlpService && this.nlpService.isEnabled() && parsed.body) {
                try {
                    const analysis = await this.nlpService.analyzeMessage(
                        parsed.uuid_v7 || 'temp',
                        parsed.body
                    );

                    // Attach NLP results to parsed record
                    parsed['nlp_markers'] = analysis.linguistic_markers;
                    parsed['word_count'] = analysis.word_count;
                    parsed['character_count'] = analysis.character_count;
                    parsed['behaviors'] = analysis.behaviors; // BehaviorMatch[]
                    parsed['entities'] = analysis.entities; // Entity[]
                } catch (nlpError) {
                    console.warn('NLP analysis failed:', nlpError);
                }
            }

            // CONVERSATION TRACKING (Group messages by participant + platform)
            if (this.supabase && this.documentId) {
                const platform = itemTag; // 'sms', 'mms', 'call'
                const participant = parsed.address || 'Unknown';
                const conversationKey = `${participant}_${platform}`;

                let conversationId = this.conversationCache.get(conversationKey);

                if (!conversationId) {
                    // Create or find conversation
                    conversationId = await this.supabase.createConversation({
                        document_id: this.documentId,
                        platform: platform,
                        participants: [participant, 'ME'],
                        start_date: parsed.date_iso || parsed.date
                    });

                    if (conversationId) {
                        this.conversationCache.set(conversationKey, conversationId);
                        console.log(`💬 Conversation created: ${conversationKey} -> ${conversationId}`);
                    }
                }

                parsed['conversation_id'] = conversationId;
                parsed['platform'] = platform;
            }

            // FILE SPLITTING LOGIC
            let shouldSplit = false;
            let splitDiscriminator = '';

            if (this.config.splitMethod === SplitMethod.DATE) {
                 const dateVal = new Date(parsed.date_iso || parsed.date);
                 let dateKey = 'UnknownDate';
                 if (!isNaN(dateVal.getTime())) {
                    const y = dateVal.getFullYear();
                    const m = String(dateVal.getMonth() + 1).padStart(2, '0');
                    dateKey = this.config.dateSplitGranularity === 'YEAR' ? `${y}` : `${y}-${m}`;
                 }
                 if (dateKey !== this.currentSplitKey) {
                     this.currentSplitKey = dateKey;
                     splitDiscriminator = dateKey;
                     shouldSplit = true;
                     this.currentSplitIndex = 1; 
                 }
            }
            else if (this.config.splitMethod === SplitMethod.ROW_COUNT && this.currentSplitRows >= this.config.splitThreshold) {
                shouldSplit = true;
            }
            else if (this.config.splitMethod === SplitMethod.FILE_SIZE_MB && (this.currentSplitSize / (1024*1024)) >= this.config.splitThreshold) {
                shouldSplit = true;
            }

            if (shouldSplit && this.config.exportLocalFile) await openNewSplit(splitDiscriminator);

            // Attachment Handling
            let exportedImageNames: string[] = [];
            let inlineBase64Parts: string[] = [];

            if (parsed._parts && parsed._parts.length > 0) {
                for (let idx = 0; idx < parsed._parts.length; idx++) {
                    const part = parsed._parts[idx];
                    let storagePath: string | null = null;
                    const ext = this.getExtension(part.ct);
                    const storageFilename = `msg_${parsed.date_iso || parsed.date}_${part.seq || 0}_${parsed.uuid_v7 || ''}.${ext}`.replace(/[:\.]/g, '_') + '.' + ext;

                    if (this.supabase && this.config.base64Option === Base64Option.UPLOAD_TO_STORAGE && this.config.storageBucket && part.data) {
                         const fileBytes = this.base64ToUint8Array(part.data);
                         if (fileBytes.length > 0) {
                             storagePath = await this.supabase.uploadFileToStorage(this.config.storageBucket, storageFilename, fileBytes, part.ct || 'application/octet-stream');
                             exportedImageNames.push(`storage://${this.config.storageBucket}/${storagePath}`);
                             part.data = '[UPLOADED_TO_STORAGE]';
                         }
                    }
                    else if (this.config.base64Option === Base64Option.EXPORT_IMAGES && imgDirHandle && part.data) {
                        this.saveImage(imgDirHandle!, storageFilename, part.data);
                        exportedImageNames.push(storageFilename);
                        part.data = '[FILE_EXPORTED]';
                    }
                    else if (this.config.base64Option === Base64Option.INLINE && part.data) {
                        inlineBase64Parts.push(part.data);
                    }
                    else if (this.config.base64Option === Base64Option.SEPARATE_FILE && partsCsvWritable && part.data) {
                        const cleanContent = part.data.replace(/[\r\n]+/g, '');
                        const escape = (s: string) => `"${(s || '').replace(/"/g, '""')}"`;
                        // CSV Format: parent_uuid,parent_date,part_seq,content_type,name,text_content,filename
                        const line = `${escape(parsed.uuid_v7)},${escape(parsed.date_iso)},${part.seq},${escape(part.ct)},${escape(part.name)},${escape(cleanContent)},${escape(storageFilename)}\n`;
                        await partsCsvWritable.write(line);
                        part.data = '[EXPORTED_TO_SEPARATE_FILE]';
                    }
                }
            }
            parsed['image_files'] = exportedImageNames.join(';');
            
            if (this.config.base64Option === Base64Option.INLINE) {
                parsed['base64_content'] = inlineBase64Parts.join('|||');
            }

            // Write CSV
            const rowStr = this.config.columns.map(h => `"${String(parsed[h] || '').replace(/"/g, '""')}"`).join(',') + '\n';
            if (csvWritable) { await csvWritable.write(rowStr); this.currentSplitSize += rowStr.length; }

            // Write SQL
            if (sqlWritable) {
                const sqlStr = this.generateSqlInsert(parsed, 'messages');
                await sqlWritable.write(sqlStr);
                this.currentSplitSize += sqlStr.length;
            }
            
            // Insert into SQLite
            if (this.sqlite) {
                this.sqlite.insert(parsed);
            }
            
            // Neo4j Graph Streaming
            if (neo4jRelsWritable && this.config.graphConfig) {
                this.config.graphConfig.nodes.forEach(nodeConfig => {
                    const idVal = parsed[nodeConfig.sourceColumn];
                    if (idVal) {
                        const props: any = {};
                        nodeConfig.properties.forEach(p => props[p] = parsed[p]);
                        if (!props['name'] && nodeConfig.label === 'Person') props['name'] = parsed['contact_name'] || idVal;
                        this.neo4jNodes.get(nodeConfig.label)?.set(idVal, props);
                    }
                });

                this.config.graphConfig.edges.forEach(async (edgeConfig) => {
                     let conditionMet = true;
                     if (edgeConfig.condition) {
                         try {
                            // eslint-disable-next-line no-new-func
                            const check = new Function('type', `return ${edgeConfig.condition}`);
                            conditionMet = check(parsed['type']);
                         } catch (e) { conditionMet = false; }
                     }

                     if (conditionMet) {
                         const from = parsed['address'] || 'Unknown';
                         const to = 'ME'; 
                         
                         const isReceived = parsed.type === '1';
                         const finalFrom = isReceived ? from : 'ME';
                         const finalTo = isReceived ? 'ME' : from;
                         
                         // Always write edge if condition met
                         await neo4jRelsWritable?.write(`${finalFrom},${finalTo},${edgeConfig.type},${parsed.date_iso},${parsed.uuid_v7}\n`);
                     }
                });
            }

            // Supabase Buffer
            if (this.supabase && this.config.streamToSupabase) {
                const record: any = {};
                this.config.columns.forEach(col => record[col] = parsed[col] ?? null);
                this.msgBuffer.push(record);
                if (this.msgBuffer.length >= SUPABASE_BATCH_SIZE) await this.flushSupabaseBuffer();
            }

            this.metadata.recordCount++;
            this.currentSplitRows++;
            if (this.metadata.recordCount % 200 === 0) onProgress(this.metadata.recordCount);
          }
        }
        
        // BUFFER MANAGEMENT FIX
        if (lastLastIndex > 0) { 
            buffer = buffer.substring(lastLastIndex); 
            itemRegex.lastIndex = 0; 
        } else if (buffer.length > CHUNK_SIZE * 2) { 
            // Only truncate if we have a massive buffer with NO matches (likely garbage or massive header)
            // But verify we don't cut in the middle of a tag if possible.
            const lastTagStart = buffer.lastIndexOf(`<${itemTag}`);
            if (lastTagStart === -1) {
                // No tag start found in 2MB+ buffer. Safe to discard beginning.
                buffer = buffer.substring(buffer.length - CHUNK_SIZE);
            } else {
                // We have a tag start, keep it and everything after
                buffer = buffer.substring(lastTagStart);
            }
        }
      }

      await this.flushSupabaseBuffer(); 
      
      // Save SQLite DB
      if (this.sqlite && outputDir) {
          const dbData = this.sqlite.export();
          const dbFile = await outputDir.getFileHandle(`${baseName}.sqlite`, { create: true });
          const dbWriter = await dbFile.createWritable();
          await dbWriter.write(dbData);
          await dbWriter.close();
          this.sqlite.close();
      }
      
      // Finalize Neo4j Nodes
      if (neo4jRelsWritable && this.config.exportNeo4j && outputDir) {
          await neo4jRelsWritable.close();
          for (const [label, rows] of this.neo4jNodes.entries()) {
              if (rows.size === 0) continue;
              const nodesFile = await outputDir.getFileHandle(`${baseName}_nodes_${label}.csv`, { create: true });
              const nodesWritable = await nodesFile.createWritable();
              const firstVal = rows.values().next().value;
              const propKeys = Object.keys(firstVal);
              await nodesWritable.write(`${label}Id:ID,${propKeys.join(',')},:LABEL\n`);
              for (const [id, props] of rows.entries()) {
                  const propVals = propKeys.map(k => `"${String(props[k] || '').replace(/"/g, '""')}"`).join(',');
                  await nodesWritable.write(`${id},${propVals},${label}\n`);
              }
              await nodesWritable.close();
          }
      }
      
      if (csvWritable) await csvWritable.close();
      if (sqlWritable) await sqlWritable.close();
      if (partsCsvWritable) await partsCsvWritable.close();
      // if (this.hasher) this.metadata.hash = this.hasher.hex(); // removed sync hash
      this.metadata.endTime = new Date().toISOString();
      if (outputDir) await this.writeReport(outputDir);

    } finally {
      // reader is done
    }
  }

  // --- Helpers ---
  private async calculateFileHash(): Promise<string | null> {
      try {
          const reader = await this.getReader();
          const hashBuffer = new Uint8Array();
          let buffer = new Uint8Array();

          while (true) {
              const { done, value } = await reader.read();
              if (done) break;
              const combined = new Uint8Array(buffer.length + value.length);
              combined.set(buffer);
              combined.set(value, buffer.length);
              buffer = combined;
          }

          const hash = await crypto.subtle.digest('SHA-256', buffer);
          const hashArray = Array.from(new Uint8Array(hash));
          return hashArray.map(b => b.toString(16).padStart(2, '0')).join('');
      } catch (e) {
          console.warn('File hash calculation failed:', e);
          return null;
      }
  }

  private async saveImage(dir: FileSystemDirectoryHandle, filename: string, base64: string) {
      try {
          const fh = await dir.getFileHandle(filename, { create: true });
          const w = await fh.createWritable();
          w.write(this.base64ToUint8Array(base64));
          w.close();
      } catch(e) {}
  }
  private async flushSupabaseBuffer() {
      if (!this.supabase || this.msgBuffer.length === 0 || !this.documentId) return;

      // Use first message's conversation ID (all messages in buffer should share same conversation in batch)
      const conversationId = this.msgBuffer[0]?.conversation_id;
      if (!conversationId) {
          console.warn('No conversation ID in buffer, skipping upload');
          return;
      }

      await this.supabase.uploadBatch(this.msgBuffer, this.documentId, conversationId);
      this.msgBuffer = [];
      this.entityBuffer.clear();
      this.attachmentBuffer = [];
  }
  private processItem(rawXml: string, itemTag: string): ParsedMessage | null {
    const result: ParsedMessage = {};
    const parser = new DOMParser();
    const doc = parser.parseFromString(rawXml, "text/xml");
    
    // Extract attributes from the main tag
    const mainNode = doc.querySelector(itemTag);
    if (mainNode && mainNode.attributes) {
        for (let i = 0; i < mainNode.attributes.length; i++) {
            const attr = mainNode.attributes[i];
            result[attr.name] = this.decodeHtmlEntity(attr.value);
        }
    } else {
        // Fallback or self-closing might be the root
        // If querySelector fails, maybe the root *is* the itemTag
        if (doc.documentElement.tagName.toLowerCase() === itemTag.toLowerCase() && doc.documentElement.attributes) {
             for (let i = 0; i < doc.documentElement.attributes.length; i++) {
                const attr = doc.documentElement.attributes[i];
                result[attr.name] = this.decodeHtmlEntity(attr.value);
            }
        }
    }

    if (!result.body) {
         // Extract text content safely
         // If there is a body attribute, we prefer that? No, usually body is attribute in SMS.
         // But in XML exports sometimes it is content. 
         // Let's take content if no attribute 'body' exists.
         if (!result['body'] && doc.documentElement.textContent) {
             result['body'] = this.decodeHtmlEntity(doc.documentElement.textContent);
         }
    }
    
    if (result.date) {
        const iso = this.dateUtils.toISO(result.date);
        if (iso) {
            result['date_iso'] = iso;
            // Remove duplicate 'date' field - keep only date_iso
            delete result['date'];
        }
        if (this.config.humanReadableTime) {
            result['date_eastern'] = this.dateUtils.toHumanEST(result.date);
        }
    }

    // Convert call type codes to human-readable (for call logs)
    if (itemTag === 'call' && result.type) {
        const typeMap: Record<string, string> = {
            '1': 'Incoming',
            '2': 'Outgoing',
            '3': 'Missed',
            '4': 'Voicemail',
            '5': 'Rejected',
            '6': 'Blocked'
        };
        result['call_type_readable'] = typeMap[result.type] || `Unknown (${result.type})`;
    }

    // Convert duration from seconds to HH:MM:SS (for call logs)
    if (result.duration) {
        const totalSeconds = parseInt(result.duration, 10);
        if (!isNaN(totalSeconds)) {
            const hours = Math.floor(totalSeconds / 3600);
            const minutes = Math.floor((totalSeconds % 3600) / 60);
            const seconds = totalSeconds % 60;
            result['duration_formatted'] = `${String(hours).padStart(2, '0')}:${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`;
        }
    }

    // Convert presentation codes to human-readable
    if (result.presentation) {
        const presentationMap: Record<string, string> = {
            '1': 'Allowed',
            '2': 'Restricted',
            '3': 'Unknown',
            '4': 'Payphone'
        };
        result['presentation_readable'] = presentationMap[result.presentation] || `Unknown (${result.presentation})`;
    }

    if (this.config.generateUuid) result['uuid_v7'] = UuidV7.generate();
    
    if (itemTag === 'mms') {
        result['_parts'] = [];
        const parts = doc.getElementsByTagName('part');
        for (let i = 0; i < parts.length; i++) {
            const part = parts[i];
            const partObj: ParsedMessage = {};
            if (part.attributes) {
                for (let j = 0; j < part.attributes.length; j++) {
                    const attr = part.attributes[j];
                    partObj[attr.name] = this.decodeHtmlEntity(attr.value);
                }
            }
            result['_parts'].push(partObj);
        }
        
        const addrs = doc.getElementsByTagName('addr');
        if (addrs.length > 0) {
            result['_addrs'] = [];
            for (let i = 0; i < addrs.length; i++) {
                 const addr = addrs[i];
                 const addrObj: ParsedMessage = {};
                 if (addr.attributes) {
                    for (let j = 0; j < addr.attributes.length; j++) {
                        const attr = addr.attributes[j];
                        addrObj[attr.name] = this.decodeHtmlEntity(attr.value);
                    }
                 }
                 result['_addrs'].push(addrObj);
            }
        }
    }
    return Object.keys(result).length > 0 ? result : null;
  }
  private async writeSqlHeader(writable: FileSystemWritableFileStream, suffix: string) {
      await writable.write(`-- SQL Dump ${suffix}\n`);
      await writable.write(`CREATE TABLE IF NOT EXISTS messages (${this.config.columns.map(c=>`"${c}" TEXT`).join(',')});\n`);
  }
  private generateSqlInsert(msg: ParsedMessage, table: string): string {
      const vals = this.config.columns.map(c => `'${String(msg[c]||'').replace(/'/g, "''")}'`);
      return `INSERT INTO ${table} (${this.config.columns.map(c=>`"${c}"`).join(',')}) VALUES (${vals.join(',')});\n`;
  }
  private async writeReport(dir: FileSystemDirectoryHandle) {
      const file = await dir.getFileHandle(`${this.metadata.originalFileName}_report.txt`, { create: true });
      const w = await file.createWritable();
      await w.write(`Report Generated: ${new Date().toISOString()}\n`);
      await w.write(`Total Records: ${this.metadata.recordCount}\n`);
      await w.write(`Validation Errors: ${this.metadata.validationErrors}\n`);
      await w.write(`Integrity Hash: ${this.metadata.hash}`);
      await w.close();
  }
  private updateMetadata(msg: ParsedMessage) {
      if (msg.contact_name) this.metadata.participants.add(msg.contact_name);
      if (msg.address) this.metadata.participants.add(msg.address);
  }
  private decodeHtmlEntity(str: string): string {
    return str.replace(/&quot;/g, '"').replace(/&apos;/g, "'").replace(/&lt;/g, '<').replace(/&gt;/g, '>').replace(/&amp;/g, '&');
  }
  private getExtension(mime: string): string {
      if (!mime) return 'bin';
      if (mime.includes('jpeg')) return 'jpg';
      if (mime.includes('png')) return 'png';
      return 'dat';
  }
  private base64ToUint8Array(base64: string): Uint8Array {
    try {
        const bin = atob(base64);
        const bytes = new Uint8Array(bin.length);
        for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i);
        return bytes;
    } catch (e) { return new Uint8Array(0); }
  }
}