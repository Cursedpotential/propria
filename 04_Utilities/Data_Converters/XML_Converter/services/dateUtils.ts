/**
 * Efficiently handles date conversions for large streams.
 * Uses Intl.DateTimeFormat instantiated once to avoid performance hits in loops.
 */
export class DateUtils {
    private estFormatter: Intl.DateTimeFormat;
  
    constructor() {
      // Setup formatter for Eastern Time (America/New_York)
      // Format: YYYY-MM-DD HH:mm:ss EST
      this.estFormatter = new Intl.DateTimeFormat('en-US', {
        timeZone: 'America/New_York',
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
        hour12: false,
        timeZoneName: 'short'
      });
    }
  
    /**
     * Converts Epoch Milliseconds (string or number) to ISO 8601
     */
    toISO(epoch: string | number): string | null {
      const ms = Number(epoch);
      if (isNaN(ms) || ms <= 0) return null;
      try {
        return new Date(ms).toISOString();
      } catch (e) {
        return null;
      }
    }
  
    /**
     * Converts Epoch Milliseconds to Human Readable EST String
     */
    toHumanEST(epoch: string | number): string | null {
      const ms = Number(epoch);
      if (isNaN(ms) || ms <= 0) return null;
      try {
        // Intl format returns parts like "10/25/2023, 14:30:00 EST"
        // We want strict sorting friendly if possible, but user asked for Human Readable.
        // Let's stick to the standard locale string for readability.
        return this.estFormatter.format(ms);
      } catch (e) {
        return null;
      }
    }
  }