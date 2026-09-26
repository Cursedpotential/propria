import pandas as pd
import re
from datetime import datetime

# First, let me analyze the complete file contents to get the full picture
all_files = []

# Let's examine what we have access to from the search results
print("File summaries based on search results:")
print("1. Conversation-analysis-request.md - Contains ChatGPT analysis framework and instructions")
print("2. abusive-language-dictionary files - Contains reference materials for abusive language patterns")
print("3. Analysis-library.md - Contains detailed behavioral analysis patterns")
print("4. Katrina-Conversation-Timeline.md - Timeline of conversations with visual components")
print("5. conversation-between-Dustin-and-Katrina-May-2024.txt - Specific conversation data")
print("6. Copy-of-sms-20221104021809.pdf - SMS backup from November 2022")
print("7. Katrina-Kinzel-5.pdf - Text message records")
print("8. ms-kk_FB_message_6.pdf - Facebook message records from August 2018")
print("9. ms-kk_FB_Katrina-Kinzel-4.pdf - Additional Facebook messages")
print("10. 18103533592.txt - Recent December 2024 conversation")

# Now let me examine the data patterns from what we can access
print("\nKey insights from available data:")
print("- Conversations span from 2018 to 2024 (6+ years)")
print("- Multiple communication platforms: SMS, Facebook, iMessage") 
print("- Evidence of systematic manipulation patterns")
print("- Child custody and coparenting conflicts")
print("- Substance abuse allegations on both sides")
print("- Financial control dynamics")
print("- Verbal abuse and derogatory language")
print("- Threats and intimidation")
print("- Patterns of blocking/unblocking communication")