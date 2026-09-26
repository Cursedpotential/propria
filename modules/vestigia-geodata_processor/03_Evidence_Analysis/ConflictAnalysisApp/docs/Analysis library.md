{  
  "patterns": {  
    "gaslighting": \[  
      "i never said that", "you imagined", "you’re imagining", "you are imagining",  
      "you’re paranoid", "that never happened", "no one will believe", "you’re crazy",  
      "you are crazy", "you’re just high", "just kidding", "you’re overreacting",  
      "this is the drugs talking"  
    \],  
    "blame\_shifting": \[  
      "this is your fault", "you made me", "because of you", "you started this",  
      "you always do this", "if you hadn’t", "look what you made me do"  
    \],  
    "minimizing": \[  
      "not a big deal", "you’re too sensitive", "calm down", "you’re being dramatic",  
      "get over it", "stop making a scene", "it was just a joke", "relax"  
    \],  
    "circular": \[  
      "what even is the point", "that’s not the point", "you keep changing",  
      "you know what i mean", "anyway", "whatever", "we’re not in high school"  
    \],  
    "contradiction": {  
      "denials": \["i don’t", "i do not", "i never", "i would never", "that’s not me", "i don’t use"\],  
      "platforms": \["snapchat", "snap", "instagram", "facebook", "tiktok"\]  
    }  
  },  
  "auxiliary": {  
    "substance\_alcohol": \["drink","drank","drunk","buzzed","tipsy","wasted","bottle","wine","beer","liquor","vodka","tequila","hungover","fireball"\],  
    "substance\_against\_you": \["crackhead","tweaker","addict","junkie","user","you’re just high","are you on something","this is the drugs talking"\],  
    "adderall\_control": \["adderall","addy","pills","script","share","split","your turn","how many did you take","i’m holding onto them for you","you can’t control yourself"\],  
    "infidelity\_places": \["huckleberry junction","huck’s","hucks"\],  
    "infidelity\_general": \["cheating","cheated","slept with","affair","secret","seeing someone","loyal","faithful","he’s just a friend","we just work together","you’re being jealous","why don’t you trust me"\],  
    "social\_platforms": \["snapchat","snap","instagram","facebook","tiktok"\],  
    "financial\_domestic": \["work","hours","job","tired","exhausted","broke","bills","rent","money","pay","afford","contribute","clean","cook","laundry","dishes","groceries","errands","lunch"\],  
    "financial\_weaponized": \["you don’t do anything","i’m the one who works hard","what do i get out of this","it’s your responsibility to provide"\],  
    "love\_bombing": \["perfect","amazing","soulmate","can’t live without you","always","forever","everything","desperate","need you","you’re the only one who understands me","i’ve never felt this way before","i want to give you everything"\],  
    "sexual\_shaming": \["slut","whore","pervert","disgusting","sick","nasty","freak","used","cheap","no wonder everyone leaves you","to think i ever did"\]  
  }  
}  
Regex tips (optional, only if you implement pre‑matching):

Case‑insensitive, word‑boundary where safe: (?i)\\blook what you made me do\\b

Slur variants (compact, careful): (?i)f\[a@\*\]g+(\[o0\])t+ (use sparingly; prefer exact quotes already present)

Platform denial pair detector: previously mentions \\b(snap|snapchat)\\b AND later (?i)\\b(i don’t|i never)\\s+(use|have)\\s+snap(chat)?\\b

5\) Starter Tag Cloud (unchanged core; aligns with your app)  
Use these tags in your UI/Search (not in the report output):

By Pattern: \#pattern:gaslighting, \#pattern:blame-shifting, \#pattern:minimizing, \#pattern:circular-argument, \#pattern:love-bombing, \#pattern:devaluation, \#pattern:hoovering

By Theme: \#theme:finances, \#theme:parenting, \#theme:isolation, \#theme:infidelity, \#theme:future-faking, \#theme:health

By Evidence Type: \#evidence:contradiction, \#evidence:admission, \#evidence:lie, \#evidence:key-quote

By Significance: \#event:major, \#event:minor, \#seed{  
  "patterns": {  
    "gaslighting": \[  
      "i never said that", "you imagined", "you’re imagining", "you are imagining",  
      "you’re paranoid", "that never happened", "no one will believe", "you’re crazy",  
      "you are crazy", "you’re just high", "just kidding", "you’re overreacting",  
      "this is the drugs talking"  
    \],  
    "blame\_shifting": \[  
      "this is your fault", "you made me", "because of you", "you started this",  
      "you always do this", "if you hadn’t", "look what you made me do"  
    \],  
    "minimizing": \[  
      "not a big deal", "you’re too sensitive", "calm down", "you’re being dramatic",  
      "get over it", "stop making a scene", "it was just a joke", "relax"  
    \],  
    "circular": \[  
      "what even is the point", "that’s not the point", "you keep changing",  
      "you know what i mean", "anyway", "whatever", "we’re not in high school"  
    \],  
    "contradiction": {  
      "denials": \["i don’t", "i do not", "i never", "i would never", "that’s not me", "i don’t use"\],  
      "platforms": \["snapchat", "snap", "instagram", "facebook", "tiktok"\]  
    }  
  },  
  "auxiliary": {  
    "substance\_alcohol": \["drink","drank","drunk","buzzed","tipsy","wasted","bottle","wine","beer","liquor","vodka","tequila","hungover","fireball"\],  
    "substance\_against\_you": \["crackhead","tweaker","addict","junkie","user","you’re just high","are you on something","this is the drugs talking"\],  
    "adderall\_control": \["adderall","addy","pills","script","share","split","your turn","how many did you take","i’m holding onto them for you","you can’t control yourself"\],  
    "infidelity\_places": \["huckleberry junction","huck’s","hucks"\],  
    "infidelity\_general": \["cheating","cheated","slept with","affair","secret","seeing someone","loyal","faithful","he’s just a friend","we just work together","you’re being jealous","why don’t you trust me"\],  
    "social\_platforms": \["snapchat","snap","instagram","facebook","tiktok"\],  
    "financial\_domestic": \["work","hours","job","tired","exhausted","broke","bills","rent","money","pay","afford","contribute","clean","cook","laundry","dishes","groceries","errands","lunch"\],  
    "financial\_weaponized": \["you don’t do anything","i’m the one who works hard","what do i get out of this","it’s your responsibility to provide"\],  
    "love\_bombing": \["perfect","amazing","soulmate","can’t live without you","always","forever","everything","desperate","need you","you’re the only one who understands me","i’ve never felt this way before","i want to give you everything"\],  
    "sexual\_shaming": \["slut","whore","pervert","disgusting","sick","nasty","freak","used","cheap","no wonder everyone leaves you","to think i ever did"\]  
  }  
}  
Regex tips (optional, only if you implement pre‑matching):

Case‑insensitive, word‑boundary where safe: (?i)\\blook what you made me do\\b

Slur variants (compact, careful): (?i)f\[a@\*\]g+(\[o0\])t+ (use sparingly; prefer exact quotes already present)

Platform denial pair detector: previously mentions \\b(snap|snapchat)\\b AND later (?i)\\b(i don’t|i never)\\s+(use|have)\\s+snap(chat)?\\b

5\) Starter Tag Cloud (unchanged core; aligns with your app)  
Use these tags in your UI/Search (not in the report output):

By Pattern: \#pattern:gaslighting, \#pattern:blame-shifting, \#pattern:minimizing, \#pattern:circular-argument, \#pattern:love-bombing, \#pattern:devaluation, \#pattern:hoovering

By Theme: \#theme:finances, \#theme:parenting, \#theme:isolation, \#theme:infidelity, \#theme:future-faking, \#theme:health

By Evidence Type: \#evidence:contradiction, \#evidence:admission, \#evidence:lie, \#evidence:key-quote

By Significance: \#event:major, \#event:minor, \#seed{  
  "patterns": {  
    "gaslighting": \[  
      "i never said that", "you imagined", "you’re imagining", "you are imagining",  
      "you’re paranoid", "that never happened", "no one will believe", "you’re crazy",  
      "you are crazy", "you’re just high", "just kidding", "you’re overreacting",  
      "this is the drugs talking"  
    \],  
    "blame\_shifting": \[  
      "this is your fault", "you made me", "because of you", "you started this",  
      "you always do this", "if you hadn’t", "look what you made me do"  
    \],  
    "minimizing": \[  
      "not a big deal", "you’re too sensitive", "calm down", "you’re being dramatic",  
      "get over it", "stop making a scene", "it was just a joke", "relax"  
    \],  
    "circular": \[  
      "what even is the point", "that’s not the point", "you keep changing",  
      "you know what i mean", "anyway", "whatever", "we’re not in high school"  
    \],  
    "contradiction": {  
      "denials": \["i don’t", "i do not", "i never", "i would never", "that’s not me", "i don’t use"\],  
      "platforms": \["snapchat", "snap", "instagram", "facebook", "tiktok"\]  
    }  
  },  
  "auxiliary": {  
    "substance\_alcohol": \["drink","drank","drunk","buzzed","tipsy","wasted","bottle","wine","beer","liquor","vodka","tequila","hungover","fireball"\],  
    "substance\_against\_you": \["crackhead","tweaker","addict","junkie","user","you’re just high","are you on something","this is the drugs talking"\],  
    "adderall\_control": \["adderall","addy","pills","script","share","split","your turn","how many did you take","i’m holding onto them for you","you can’t control yourself"\],  
    "infidelity\_places": \["huckleberry junction","huck’s","hucks"\],  
    "infidelity\_general": \["cheating","cheated","slept with","affair","secret","seeing someone","loyal","faithful","he’s just a friend","we just work together","you’re being jealous","why don’t you trust me"\],  
    "social\_platforms": \["snapchat","snap","instagram","facebook","tiktok"\],  
    "financial\_domestic": \["work","hours","job","tired","exhausted","broke","bills","rent","money","pay","afford","contribute","clean","cook","laundry","dishes","groceries","errands","lunch"\],  
    "financial\_weaponized": \["you don’t do anything","i’m the one who works hard","what do i get out of this","it’s your responsibility to provide"\],  
    "love\_bombing": \["perfect","amazing","soulmate","can’t live without you","always","forever","everything","desperate","need you","you’re the only one who understands me","i’ve never felt this way before","i want to give you everything"\],  
    "sexual\_shaming": \["slut","whore","pervert","disgusting","sick","nasty","freak","used","cheap","no wonder everyone leaves you","to think i ever did"\]  
  }  
}  
Regex tips (optional, only if you implement pre‑matching):

Case‑insensitive, word‑boundary where safe: (?i)\\blook what you made me do\\b

Slur variants (compact, careful): (?i)f\[a@\*\]g+(\[o0\])t+ (use sparingly; prefer exact quotes already present)

Platform denial pair detector: previously mentions \\b(snap|snapchat)\\b AND later (?i)\\b(i don’t|i never)\\s+(use|have)\\s+snap(chat)?\\b

5\) Starter Tag Cloud (unchanged core; aligns with your app)  
Use these tags in your UI/Search (not in the report output):

By Pattern: \#pattern:gaslighting, \#pattern:blame-shifting, \#pattern:minimizing, \#pattern:circular-argument, \#pattern:love-bombing, \#pattern:devaluation, \#pattern:hoovering

By Theme: \#theme:finances, \#theme:parenting, \#theme:isolation, \#theme:infidelity, \#theme:future-faking, \#theme:health

By Evidence Type: \#evidence:contradiction, \#evidence:admission, \#evidence:lie, \#evidence:key-quote

By Significance: \#event:major, \#event:minor, \#seed{  
  "patterns": {  
    "gaslighting": \[  
      "i never said that", "you imagined", "you’re imagining", "you are imagining",  
      "you’re paranoid", "that never happened", "no one will believe", "you’re crazy",  
      "you are crazy", "you’re just high", "just kidding", "you’re overreacting",  
      "this is the drugs talking"  
    \],  
    "blame\_shifting": \[  
      "this is your fault", "you made me", "because of you", "you started this",  
      "you always do this", "if you hadn’t", "look what you made me do"  
    \],  
    "minimizing": \[  
      "not a big deal", "you’re too sensitive", "calm down", "you’re being dramatic",  
      "get over it", "stop making a scene", "it was just a joke", "relax"  
    \],  
    "circular": \[  
      "what even is the point", "that’s not the point", "you keep changing",  
      "you know what i mean", "anyway", "whatever", "we’re not in high school"  
    \],  
    "contradiction": {  
      "denials": \["i don’t", "i do not", "i never", "i would never", "that’s not me", "i don’t use"\],  
      "platforms": \["snapchat", "snap", "instagram", "facebook", "tiktok"\]  
    }  
  },  
  "auxiliary": {  
    "substance\_alcohol": \["drink","drank","drunk","buzzed","tipsy","wasted","bottle","wine","beer","liquor","vodka","tequila","hungover","fireball"\],  
    "substance\_against\_you": \["crackhead","tweaker","addict","junkie","user","you’re just high","are you on something","this is the drugs talking"\],  
    "adderall\_control": \["adderall","addy","pills","script","share","split","your turn","how many did you take","i’m holding onto them for you","you can’t control yourself"\],  
    "infidelity\_places": \["huckleberry junction","huck’s","hucks"\],  
    "infidelity\_general": \["cheating","cheated","slept with","affair","secret","seeing someone","loyal","faithful","he’s just a friend","we just work together","you’re being jealous","why don’t you trust me"\],  
    "social\_platforms": \["snapchat","snap","instagram","facebook","tiktok"\],  
    "financial\_domestic": \["work","hours","job","tired","exhausted","broke","bills","rent","money","pay","afford","contribute","clean","cook","laundry","dishes","groceries","errands","lunch"\],  
    "financial\_weaponized": \["you don’t do anything","i’m the one who works hard","what do i get out of this","it’s your responsibility to provide"\],  
    "love\_bombing": \["perfect","amazing","soulmate","can’t live without you","always","forever","everything","desperate","need you","you’re the only one who understands me","i’ve never felt this way before","i want to give you everything"\],  
    "sexual\_shaming": \["slut","whore","pervert","disgusting","sick","nasty","freak","used","cheap","no wonder everyone leaves you","to think i ever did"\]  
  }  
}  
Regex tips (optional, only if you implement pre‑matching):

Case‑insensitive, word‑boundary where safe: (?i)\\blook what you made me do\\b

Slur variants (compact, careful): (?i)f\[a@\*\]g+(\[o0\])t+ (use sparingly; prefer exact quotes already present)

Platform denial pair detector: previously mentions \\b(snap|snapchat)\\b AND later (?i)\\b(i don’t|i never)\\s+(use|have)\\s+snap(chat)?\\b

5\) Starter Tag Cloud (unchanged core; aligns with your app)  
Use these tags in your UI/Search (not in the report output):

By Pattern: \#pattern:gaslighting, \#pattern:blame-shifting, \#pattern:minimizing, \#pattern:circular-argument, \#pattern:love-bombing, \#pattern:devaluation, \#pattern:hoovering

By Theme: \#theme:finances, \#theme:parenting, \#theme:isolation, \#theme:infidelity, \#theme:future-faking, \#theme:health

By Evidence Type: \#evidence:contradiction, \#evidence:admission, \#evidence:lie, \#evidence:key-quote

By Significance: \#event:major, \#event:minor, \#seed{  
  "patterns": {  
    "gaslighting": \[  
      "i never said that", "you imagined", "you’re imagining", "you are imagining",  
      "you’re paranoid", "that never happened", "no one will believe", "you’re crazy",  
      "you are crazy", "you’re just high", "just kidding", "you’re overreacting",  
      "this is the drugs talking"  
    \],  
    "blame\_shifting": \[  
      "this is your fault", "you made me", "because of you", "you started this",  
      "you always do this", "if you hadn’t", "look what you made me do"  
    \],  
    "minimizing": \[  
      "not a big deal", "you’re too sensitive", "calm down", "you’re being dramatic",  
      "get over it", "stop making a scene", "it was just a joke", "relax"  
    \],  
    "circular": \[  
      "what even is the point", "that’s not the point", "you keep changing",  
      "you know what i mean", "anyway", "whatever", "we’re not in high school"  
    \],  
    "contradiction": {  
      "denials": \["i don’t", "i do not", "i never", "i would never", "that’s not me", "i don’t use"\],  
      "platforms": \["snapchat", "snap", "instagram", "facebook", "tiktok"\]  
    }  
  },  
  "auxiliary": {  
    "substance\_alcohol": \["drink","drank","drunk","buzzed","tipsy","wasted","bottle","wine","beer","liquor","vodka","tequila","hungover","fireball"\],  
    "substance\_against\_you": \["crackhead","tweaker","addict","junkie","user","you’re just high","are you on something","this is the drugs talking"\],  
    "adderall\_control": \["adderall","addy","pills","script","share","split","your turn","how many did you take","i’m holding onto them for you","you can’t control yourself"\],  
    "infidelity\_places": \["huckleberry junction","huck’s","hucks"\],  
    "infidelity\_general": \["cheating","cheated","slept with","affair","secret","seeing someone","loyal","faithful","he’s just a friend","we just work together","you’re being jealous","why don’t you trust me"\],  
    "social\_platforms": \["snapchat","snap","instagram","facebook","tiktok"\],  
    "financial\_domestic": \["work","hours","job","tired","exhausted","broke","bills","rent","money","pay","afford","contribute","clean","cook","laundry","dishes","groceries","errands","lunch"\],  
    "financial\_weaponized": \["you don’t do anything","i’m the one who works hard","what do i get out of this","it’s your responsibility to provide"\],  
    "love\_bombing": \["perfect","amazing","soulmate","can’t live without you","always","forever","everything","desperate","need you","you’re the only one who understands me","i’ve never felt this way before","i want to give you everything"\],  
    "sexual\_shaming": \["slut","whore","pervert","disgusting","sick","nasty","freak","used","cheap","no wonder everyone leaves you","to think i ever did"\]  
  }  
}  
Regex tips (optional, only if you implement pre‑matching):

Case‑insensitive, word‑boundary where safe: (?i)\\blook what you made me do\\b

Slur variants (compact, careful): (?i)f\[a@\*\]g+(\[o0\])t+ (use sparingly; prefer exact quotes already present)

Platform denial pair detector: previously mentions \\b(snap|snapchat)\\b AND later (?i)\\b(i don’t|i never)\\s+(use|have)\\s+snap(chat)?\\b

5\) Starter Tag Cloud (unchanged core; aligns with your app)  
Use these tags in your UI/Search (not in the report output):

By Pattern: \#pattern:gaslighting, \#pattern:blame-shifting, \#pattern:minimizing, \#pattern:circular-argument, \#pattern:love-bombing, \#pattern:devaluation, \#pattern:hoovering

By Theme: \#theme:finances, \#theme:parenting, \#theme:isolation, \#theme:infidelity, \#theme:future-faking, \#theme:health

By Evidence Type: \#evidence:contradiction, \#evidence:admission, \#evidence:lie, \#evidence:key-quote

By Significance: \#event:major, \#event:minor, \#seed{  
  "patterns": {  
    "gaslighting": \[  
      "i never said that", "you imagined", "you’re imagining", "you are imagining",  
      "you’re paranoid", "that never happened", "no one will believe", "you’re crazy",  
      "you are crazy", "you’re just high", "just kidding", "you’re overreacting",  
      "this is the drugs talking"  
    \],  
    "blame\_shifting": \[  
      "this is your fault", "you made me", "because of you", "you started this",  
      "you always do this", "if you hadn’t", "look what you made me do"  
    \],  
    "minimizing": \[  
      "not a big deal", "you’re too sensitive", "calm down", "you’re being dramatic",  
      "get over it", "stop making a scene", "it was just a joke", "relax"  
    \],  
    "circular": \[  
      "what even is the point", "that’s not the point", "you keep changing",  
      "you know what i mean", "anyway", "whatever", "we’re not in high school"  
    \],  
    "contradiction": {  
      "denials": \["i don’t", "i do not", "i never", "i would never", "that’s not me", "i don’t use"\],  
      "platforms": \["snapchat", "snap", "instagram", "facebook", "tiktok"\]  
    }  
  },  
  "auxiliary": {  
    "substance\_alcohol": \["drink","drank","drunk","buzzed","tipsy","wasted","bottle","wine","beer","liquor","vodka","tequila","hungover","fireball"\],  
    "substance\_against\_you": \["crackhead","tweaker","addict","junkie","user","you’re just high","are you on something","this is the drugs talking"\],  
    "adderall\_control": \["adderall","addy","pills","script","share","split","your turn","how many did you take","i’m holding onto them for you","you can’t control yourself"\],  
    "infidelity\_places": \["huckleberry junction","huck’s","hucks"\],  
    "infidelity\_general": \["cheating","cheated","slept with","affair","secret","seeing someone","loyal","faithful","he’s just a friend","we just work together","you’re being jealous","why don’t you trust me"\],  
    "social\_platforms": \["snapchat","snap","instagram","facebook","tiktok"\],  
    "financial\_domestic": \["work","hours","job","tired","exhausted","broke","bills","rent","money","pay","afford","contribute","clean","cook","laundry","dishes","groceries","errands","lunch"\],  
    "financial\_weaponized": \["you don’t do anything","i’m the one who works hard","what do i get out of this","it’s your responsibility to provide"\],  
    "love\_bombing": \["perfect","amazing","soulmate","can’t live without you","always","forever","everything","desperate","need you","you’re the only one who understands me","i’ve never felt this way before","i want to give you everything"\],  
    "sexual\_shaming": \["slut","whore","pervert","disgusting","sick","nasty","freak","used","cheap","no wonder everyone leaves you","to think i ever did"\]  
  }  
}  
Regex tips (optional, only if you implement pre‑matching):

Case‑insensitive, word‑boundary where safe: (?i)\\blook what you made me do\\b

Slur variants (compact, careful): (?i)f\[a@\*\]g+(\[o0\])t+ (use sparingly; prefer exact quotes already present)

Platform denial pair detector: previously mentions \\b(snap|snapchat)\\b AND later (?i)\\b(i don’t|i never)\\s+(use|have)\\s+snap(chat)?\\b

5\) Starter Tag Cloud (unchanged core; aligns with your app)  
Use these tags in your UI/Search (not in the report output):

By Pattern: \#pattern:gaslighting, \#pattern:blame-shifting, \#pattern:minimizing, \#pattern:circular-argument, \#pattern:love-bombing, \#pattern:devaluation, \#pattern:hoovering

By Theme: \#theme:finances, \#theme:parenting, \#theme:isolation, \#theme:infidelity, \#theme:future-faking, \#theme:health

By Evidence Type: \#evidence:contradiction, \#evidence:admission, \#evidence:lie, \#evidence:key-quote

By Significance: \#event:major, \#event:minor, \#seed{  
  "patterns": {  
    "gaslighting": \[  
      "i never said that", "you imagined", "you’re imagining", "you are imagining",  
      "you’re paranoid", "that never happened", "no one will believe", "you’re crazy",  
      "you are crazy", "you’re just high", "just kidding", "you’re overreacting",  
      "this is the drugs talking"  
    \],  
    "blame\_shifting": \[  
      "this is your fault", "you made me", "because of you", "you started this",  
      "you always do this", "if you hadn’t", "look what you made me do"  
    \],  
    "minimizing": \[  
      "not a big deal", "you’re too sensitive", "calm down", "you’re being dramatic",  
      "get over it", "stop making a scene", "it was just a joke", "relax"  
    \],  
    "circular": \[  
      "what even is the point", "that’s not the point", "you keep changing",  
      "you know what i mean", "anyway", "whatever", "we’re not in high school"  
    \],  
    "contradiction": {  
      "denials": \["i don’t", "i do not", "i never", "i would never", "that’s not me", "i don’t use"\],  
      "platforms": \["snapchat", "snap", "instagram", "facebook", "tiktok"\]  
    }  
  },  
  "auxiliary": {  
    "substance\_alcohol": \["drink","drank","drunk","buzzed","tipsy","wasted","bottle","wine","beer","liquor","vodka","tequila","hungover","fireball"\],  
    "substance\_against\_you": \["crackhead","tweaker","addict","junkie","user","you’re just high","are you on something","this is the drugs talking"\],  
    "adderall\_control": \["adderall","addy","pills","script","share","split","your turn","how many did you take","i’m holding onto them for you","you can’t control yourself"\],  
    "infidelity\_places": \["huckleberry junction","huck’s","hucks"\],  
    "infidelity\_general": \["cheating","cheated","slept with","affair","secret","seeing someone","loyal","faithful","he’s just a friend","we just work together","you’re being jealous","why don’t you trust me"\],  
    "social\_platforms": \["snapchat","snap","instagram","facebook","tiktok"\],  
    "financial\_domestic": \["work","hours","job","tired","exhausted","broke","bills","rent","money","pay","afford","contribute","clean","cook","laundry","dishes","groceries","errands","lunch"\],  
    "financial\_weaponized": \["you don’t do anything","i’m the one who works hard","what do i get out of this","it’s your responsibility to provide"\],  
    "love\_bombing": \["perfect","amazing","soulmate","can’t live without you","always","forever","everything","desperate","need you","you’re the only one who understands me","i’ve never felt this way before","i want to give you everything"\],  
    "sexual\_shaming": \["slut","whore","pervert","disgusting","sick","nasty","freak","used","cheap","no wonder everyone leaves you","to think i ever did"\]  
  }  
}  
Regex tips (optional, only if you implement pre‑matching):

Case‑insensitive, word‑boundary where safe: (?i)\\blook what you made me do\\b

Slur variants (compact, careful): (?i)f\[a@\*\]g+(\[o0\])t+ (use sparingly; prefer exact quotes already present)

Platform denial pair detector: previously mentions \\b(snap|snapchat)\\b AND later (?i)\\b(i don’t|i never)\\s+(use|have)\\s+snap(chat)?\\b

5\) Starter Tag Cloud (unchanged core; aligns with your app)  
Use these tags in your UI/Search (not in the report output):

By Pattern: \#pattern:gaslighting, \#pattern:blame-shifting, \#pattern:minimizing, \#pattern:circular-argument, \#pattern:love-bombing, \#pattern:devaluation, \#pattern:hoovering

By Theme: \#theme:finances, \#theme:parenting, \#theme:isolation, \#theme:infidelity, \#theme:future-faking, \#theme:health

By Evidence Type: \#evidence:contradiction, \#evidence:admission, \#evidence:lie, \#evidence:key-quote

By Significance: \#event:major, \#event:minor, \#seed{  
  "patterns": {  
    "gaslighting": \[  
      "i never said that", "you imagined", "you’re imagining", "you are imagining",  
      "you’re paranoid", "that never happened", "no one will believe", "you’re crazy",  
      "you are crazy", "you’re just high", "just kidding", "you’re overreacting",  
      "this is the drugs talking"  
    \],  
    "blame\_shifting": \[  
      "this is your fault", "you made me", "because of you", "you started this",  
      "you always do this", "if you hadn’t", "look what you made me do"  
    \],  
    "minimizing": \[  
      "not a big deal", "you’re too sensitive", "calm down", "you’re being dramatic",  
      "get over it", "stop making a scene", "it was just a joke", "relax"  
    \],  
    "circular": \[  
      "what even is the point", "that’s not the point", "you keep changing",  
      "you know what i mean", "anyway", "whatever", "we’re not in high school"  
    \],  
    "contradiction": {  
      "denials": \["i don’t", "i do not", "i never", "i would never", "that’s not me", "i don’t use"\],  
      "platforms": \["snapchat", "snap", "instagram", "facebook", "tiktok"\]  
    }  
  },  
  "auxiliary": {  
    "substance\_alcohol": \["drink","drank","drunk","buzzed","tipsy","wasted","bottle","wine","beer","liquor","vodka","tequila","hungover","fireball"\],  
    "substance\_against\_you": \["crackhead","tweaker","addict","junkie","user","you’re just high","are you on something","this is the drugs talking"\],  
    "adderall\_control": \["adderall","addy","pills","script","share","split","your turn","how many did you take","i’m holding onto them for you","you can’t control yourself"\],  
    "infidelity\_places": \["huckleberry junction","huck’s","hucks"\],  
    "infidelity\_general": \["cheating","cheated","slept with","affair","secret","seeing someone","loyal","faithful","he’s just a friend","we just work together","you’re being jealous","why don’t you trust me"\],  
    "social\_platforms": \["snapchat","snap","instagram","facebook","tiktok"\],  
    "financial\_domestic": \["work","hours","job","tired","exhausted","broke","bills","rent","money","pay","afford","contribute","clean","cook","laundry","dishes","groceries","errands","lunch"\],  
    "financial\_weaponized": \["you don’t do anything","i’m the one who works hard","what do i get out of this","it’s your responsibility to provide"\],  
    "love\_bombing": \["perfect","amazing","soulmate","can’t live without you","always","forever","everything","desperate","need you","you’re the only one who understands me","i’ve never felt this way before","i want to give you everything"\],  
    "sexual\_shaming": \["slut","whore","pervert","disgusting","sick","nasty","freak","used","cheap","no wonder everyone leaves you","to think i ever did"\]  
  }  
}  
Regex tips (optional, only if you implement pre‑matching):

Case‑insensitive, word‑boundary where safe: (?i)\\blook what you made me do\\b

Slur variants (compact, careful): (?i)f\[a@\*\]g+(\[o0\])t+ (use sparingly; prefer exact quotes already present)

Platform denial pair detector: previously mentions \\b(snap|snapchat)\\b AND later (?i)\\b(i don’t|i never)\\s+(use|have)\\s+snap(chat)?\\b

5\) Starter Tag Cloud (unchanged core; aligns with your app)  
Use these tags in your UI/Search (not in the report output):

By Pattern: \#pattern:gaslighting, \#pattern:blame-shifting, \#pattern:minimizing, \#pattern:circular-argument, \#pattern:love-bombing, \#pattern:devaluation, \#pattern:hoovering

By Theme: \#theme:finances, \#theme:parenting, \#theme:isolation, \#theme:infidelity, \#theme:future-faking, \#theme:health

By Evidence Type: \#evidence:contradiction, \#evidence:admission, \#evidence:lie, \#evidence:key-quote

By Significance: \#event:major, \#event:minor, \#seed{  
  "patterns": {  
    "gaslighting": \[  
      "i never said that", "you imagined", "you’re imagining", "you are imagining",  
      "you’re paranoid", "that never happened", "no one will believe", "you’re crazy",  
      "you are crazy", "you’re just high", "just kidding", "you’re overreacting",  
      "this is the drugs talking"  
    \],  
    "blame\_shifting": \[  
      "this is your fault", "you made me", "because of you", "you started this",  
      "you always do this", "if you hadn’t", "look what you made me do"  
    \],  
    "minimizing": \[  
      "not a big deal", "you’re too sensitive", "calm down", "you’re being dramatic",  
      "get over it", "stop making a scene", "it was just a joke", "relax"  
    \],  
    "circular": \[  
      "what even is the point", "that’s not the point", "you keep changing",  
      "you know what i mean", "anyway", "whatever", "we’re not in high school"  
    \],  
    "contradiction": {  
      "denials": \["i don’t", "i do not", "i never", "i would never", "that’s not me", "i don’t use"\],  
      "platforms": \["snapchat", "snap", "instagram", "facebook", "tiktok"\]  
    }  
  },  
  "auxiliary": {  
    "substance\_alcohol": \["drink","drank","drunk","buzzed","tipsy","wasted","bottle","wine","beer","liquor","vodka","tequila","hungover","fireball"\],  
    "substance\_against\_you": \["crackhead","tweaker","addict","junkie","user","you’re just high","are you on something","this is the drugs talking"\],  
    "adderall\_control": \["adderall","addy","pills","script","share","split","your turn","how many did you take","i’m holding onto them for you","you can’t control yourself"\],  
    "infidelity\_places": \["huckleberry junction","huck’s","hucks"\],  
    "infidelity\_general": \["cheating","cheated","slept with","affair","secret","seeing someone","loyal","faithful","he’s just a friend","we just work together","you’re being jealous","why don’t you trust me"\],  
    "social\_platforms": \["snapchat","snap","instagram","facebook","tiktok"\],  
    "financial\_domestic": \["work","hours","job","tired","exhausted","broke","bills","rent","money","pay","afford","contribute","clean","cook","laundry","dishes","groceries","errands","lunch"\],  
    "financial\_weaponized": \["you don’t do anything","i’m the one who works hard","what do i get out of this","it’s your responsibility to provide"\],  
    "love\_bombing": \["perfect","amazing","soulmate","can’t live without you","always","forever","everything","desperate","need you","you’re the only one who understands me","i’ve never felt this way before","i want to give you everything"\],  
    "sexual\_shaming": \["slut","whore","pervert","disgusting","sick","nasty","freak","used","cheap","no wonder everyone leaves you","to think i ever did"\]  
  }  
}  
Regex tips (optional, only if you implement pre‑matching):

Case‑insensitive, word‑boundary where safe: (?i)\\blook what you made me do\\b

Slur variants (compact, careful): (?i)f\[a@\*\]g+(\[o0\])t+ (use sparingly; prefer exact quotes already present)

Platform denial pair detector: previously mentions \\b(snap|snapchat)\\b AND later (?i)\\b(i don’t|i never)\\s+(use|have)\\s+snap(chat)?\\b

5\) Starter Tag Cloud (unchanged core; aligns with your app)  
Use these tags in your UI/Search (not in the report output):

By Pattern: \#pattern:gaslighting, \#pattern:blame-shifting, \#pattern:minimizing, \#pattern:circular-argument, \#pattern:love-bombing, \#pattern:devaluation, \#pattern:hoovering

By Theme: \#theme:finances, \#theme:parenting, \#theme:isolation, \#theme:infidelity, \#theme:future-faking, \#theme:health

By Evidence Type: \#evidence:contradiction, \#evidence:admission, \#evidence:lie, \#evidence:key-quote

By Significance: \#event:major, \#event:minor, \#seed{  
  "patterns": {  
    "gaslighting": \[  
      "i never said that", "you imagined", "you’re imagining", "you are imagining",  
      "you’re paranoid", "that never happened", "no one will believe", "you’re crazy",  
      "you are crazy", "you’re just high", "just kidding", "you’re overreacting",  
      "this is the drugs talking"  
    \],  
    "blame\_shifting": \[  
      "this is your fault", "you made me", "because of you", "you started this",  
      "you always do this", "if you hadn’t", "look what you made me do"  
    \],  
    "minimizing": \[  
      "not a big deal", "you’re too sensitive", "calm down", "you’re being dramatic",  
      "get over it", "stop making a scene", "it was just a joke", "relax"  
    \],  
    "circular": \[  
      "what even is the point", "that’s not the point", "you keep changing",  
      "you know what i mean", "anyway", "whatever", "we’re not in high school"  
    \],  
    "contradiction": {  
      "denials": \["i don’t", "i do not", "i never", "i would never", "that’s not me", "i don’t use"\],  
      "platforms": \["snapchat", "snap", "instagram", "facebook", "tiktok"\]  
    }  
  },  
  "auxiliary": {  
    "substance\_alcohol": \["drink","drank","drunk","buzzed","tipsy","wasted","bottle","wine","beer","liquor","vodka","tequila","hungover","fireball"\],  
    "substance\_against\_you": \["crackhead","tweaker","addict","junkie","user","you’re just high","are you on something","this is the drugs talking"\],  
    "adderall\_control": \["adderall","addy","pills","script","share","split","your turn","how many did you take","i’m holding onto them for you","you can’t control yourself"\],  
    "infidelity\_places": \["huckleberry junction","huck’s","hucks"\],  
    "infidelity\_general": \["cheating","cheated","slept with","affair","secret","seeing someone","loyal","faithful","he’s just a friend","we just work together","you’re being jealous","why don’t you trust me"\],  
    "social\_platforms": \["snapchat","snap","instagram","facebook","tiktok"\],  
    "financial\_domestic": \["work","hours","job","tired","exhausted","broke","bills","rent","money","pay","afford","contribute","clean","cook","laundry","dishes","groceries","errands","lunch"\],  
    "financial\_weaponized": \["you don’t do anything","i’m the one who works hard","what do i get out of this","it’s your responsibility to provide"\],  
    "love\_bombing": \["perfect","amazing","soulmate","can’t live without you","always","forever","everything","desperate","need you","you’re the only one who understands me","i’ve never felt this way before","i want to give you everything"\],  
    "sexual\_shaming": \["slut","whore","pervert","disgusting","sick","nasty","freak","used","cheap","no wonder everyone leaves you","to think i ever did"\]  
  }  
}  
Regex tips (optional, only if you implement pre‑matching):

Case‑insensitive, word‑boundary where safe: (?i)\\blook what you made me do\\b

Slur variants (compact, careful): (?i)f\[a@\*\]g+(\[o0\])t+ (use sparingly; prefer exact quotes already present)

Platform denial pair detector: previously mentions \\b(snap|snapchat)\\b AND later (?i)\\b(i don’t|i never)\\s+(use|have)\\s+snap(chat)?\\b

5\) Starter Tag Cloud (unchanged core; aligns with your app)  
Use these tags in your UI/Search (not in the report output):

By Pattern: \#pattern:gaslighting, \#pattern:blame-shifting, \#pattern:minimizing, \#pattern:circular-argument, \#pattern:love-bombing, \#pattern:devaluation, \#pattern:hoovering

By Theme: \#theme:finances, \#theme:parenting, \#theme:isolation, \#theme:infidelity, \#theme:future-faking, \#theme:health

By Evidence Type: \#evidence:contradiction, \#evidence:admission, \#evidence:lie, \#evidence:key-quote

By Significance: \#event:major, \#event:minor, \#seed{  
  "patterns": {  
    "gaslighting": \[  
      "i never said that", "you imagined", "you’re imagining", "you are imagining",  
      "you’re paranoid", "that never happened", "no one will believe", "you’re crazy",  
      "you are crazy", "you’re just high", "just kidding", "you’re overreacting",  
      "this is the drugs talking"  
    \],  
    "blame\_shifting": \[  
      "this is your fault", "you made me", "because of you", "you started this",  
      "you always do this", "if you hadn’t", "look what you made me do"  
    \],  
    "minimizing": \[  
      "not a big deal", "you’re too sensitive", "calm down", "you’re being dramatic",  
      "get over it", "stop making a scene", "it was just a joke", "relax"  
    \],  
    "circular": \[  
      "what even is the point", "that’s not the point", "you keep changing",  
      "you know what i mean", "anyway", "whatever", "we’re not in high school"  
    \],  
    "contradiction": {  
      "denials": \["i don’t", "i do not", "i never", "i would never", "that’s not me", "i don’t use"\],  
      "platforms": \["snapchat", "snap", "instagram", "facebook", "tiktok"\]  
    }  
  },  
  "auxiliary": {  
    "substance\_alcohol": \["drink","drank","drunk","buzzed","tipsy","wasted","bottle","wine","beer","liquor","vodka","tequila","hungover","fireball"\],  
    "substance\_against\_you": \["crackhead","tweaker","addict","junkie","user","you’re just high","are you on something","this is the drugs talking"\],  
    "adderall\_control": \["adderall","addy","pills","script","share","split","your turn","how many did you take","i’m holding onto them for you","you can’t control yourself"\],  
    "infidelity\_places": \["huckleberry junction","huck’s","hucks"\],  
    "infidelity\_general": \["cheating","cheated","slept with","affair","secret","seeing someone","loyal","faithful","he’s just a friend","we just work together","you’re being jealous","why don’t you trust me"\],  
    "social\_platforms": \["snapchat","snap","instagram","facebook","tiktok"\],  
    "financial\_domestic": \["work","hours","job","tired","exhausted","broke","bills","rent","money","pay","afford","contribute","clean","cook","laundry","dishes","groceries","errands","lunch"\],  
    "financial\_weaponized": \["you don’t do anything","i’m the one who works hard","what do i get out of this","it’s your responsibility to provide"\],  
    "love\_bombing": \["perfect","amazing","soulmate","can’t live without you","always","forever","everything","desperate","need you","you’re the only one who understands me","i’ve never felt this way before","i want to give you everything"\],  
    "sexual\_shaming": \["slut","whore","pervert","disgusting","sick","nasty","freak","used","cheap","no wonder everyone leaves you","to think i ever did"\]  
  }  
}  
Regex tips (optional, only if you implement pre‑matching):

Case‑insensitive, word‑boundary where safe: (?i)\\blook what you made me do\\b

Slur variants (compact, careful): (?i)f\[a@\*\]g+(\[o0\])t+ (use sparingly; prefer exact quotes already present)

Platform denial pair detector: previously mentions \\b(snap|snapchat)\\b AND later (?i)\\b(i don’t|i never)\\s+(use|have)\\s+snap(chat)?\\b

5\) Starter Tag Cloud (unchanged core; aligns with your app)  
Use these tags in your UI/Search (not in the report output):

By Pattern: \#pattern:gaslighting, \#pattern:blame-shifting, \#pattern:minimizing, \#pattern:circular-argument, \#pattern:love-bombing, \#pattern:devaluation, \#pattern:hoovering

By Theme: \#theme:finances, \#theme:parenting, \#theme:isolation, \#theme:infidelity, \#theme:future-faking, \#theme:health

By Evidence Type: \#evidence:contradiction, \#evidence:admission, \#evidence:lie, \#evidence:key-quote

By Significance: \#event:major, \#event:minor, \#seed{  
  "substance\_abuse": {  
    "version": "1.0",  
    "description": "Substance abuse and weaponization patterns",  
    "categories": {  
      "alcohol\_abuse": {  
        "keywords": \[  
          "drink",  
          "drank",  
          "drunk",  
          "buzzed",  
          "tipsy",  
          "wasted",  
          "bottle",  
          "wine",  
          "beer",  
          "liquor",  
          "vodka",  
          "tequila",  
          "hungover",  
          "fireball"  
        \],  
        "risk\_phrases": \[  
          "I only had one",  
          "I'm fine to drive",  
          "I can handle it",  
          "Stop worrying",  
          "It's not a big deal",  
          "I need to relax"  
        \],  
        "child\_endangerment\_flags": \[  
          "drive",  
          "driving",  
          "car",  
          "pick up",  
          "daycare",  
          "school"  
        \]  
      },  
      "weaponized\_accusations": {  
        "insults": \[  
          "crackhead",  
          "tweaker",  
          "addict",  
          "junkie",  
          "user"  
        \],  
        "dismissal\_phrases": \[  
          "You're just high",  
          "Are you on something right now",  
          "This is the drugs talking",  
          "No one will believe a tweaker",  
          "You're paranoid"  
        \]  
      },  
      "adderall\_control": {  
        "keywords": \[  
          "Adderall",  
          "addy",  
          "pills",  
          "script",  
          "share",  
          "split",  
          "your turn"  
        \],  
        "control\_phrases": \[  
          "How many did you take",  
          "You're taking too many",  
          "I only took one",  
          "I'm holding onto them for you",  
          "You can't control yourself"  
        \]  
      }  
    }  
  },  
  "deception\_control": {  
    "version": "1.0",  
    "description": "Deception and narrative control patterns",  
    "categories": {  
      "gaslighting": {  
        "keywords": \[  
          "crazy",  
          "sensitive",  
          "overreacting",  
          "imagining",  
          "wrong",  
          "never happened",  
          "didn't say that",  
          "forgetting",  
          "confused",  
          "misremembering"  
        \],  
        "phrases": \[  
          "You're being dramatic",  
          "That's not how it happened",  
          "I was just kidding",  
          "You're twisting my words",  
          "You have issues"  
        \]  
      },  
      "character\_assassination": {  
        "severe\_abuse": \[  
          "bitch made",  
          "motherfucker",  
          "piece of shit",  
          "good for nothing",  
          "ain't shit"  
        \],  
        "homophobic\_slurs": \[  
          "fagot",  
          "fagget",  
          "faggot",  
          "sick fagot"  
        \],  
        "mental\_health\_weapons": \[  
          "stupid",  
          "sick",  
          "weirdo",  
          "psycho",  
          "loser",  
          "freak",  
          "pathetic",  
          "unstable"  
        \]  
      },  
      "infidelity\_betrayal": {  
        "locations": \[  
          "Huckleberry Junction",  
          "Huck's"  
        \],  
        "keywords": \[  
          "cheating",  
          "cheated",  
          "slept with",  
          "affair",  
          "secret",  
          "seeing someone",  
          "loyal",  
          "faithful"  
        \],  
        "suspicious\_phrases": \[  
          "He's just a friend",  
          "We just work together",  
          "You're being jealous",  
          "Why don't you trust me"  
        \]  
      },  
      "social\_media\_deception": {  
        "platforms": \[  
          "Snapchat",  
          "Snap",  
          "social media",  
          "Instagram",  
          "Facebook",  
          "TikTok"  
        \],  
        "deceptive\_phrases": \[  
          "I don't even use Snapchat",  
          "I never send pictures",  
          "You can check my phone"  
        \],  
        "salacious\_content": \[  
          "nudes",  
          "pics",  
          "pictures",  
          "sexy",  
          "hot",  
          "videos",  
          "selfie"  
        \]  
      },  
      "feigning\_incompetence": {  
        "keywords": \[  
          "dumb",  
          "stupid",  
          "don't know how",  
          "confused",  
          "don't understand",  
          "forgot",  
          "can't remember"  
        \],  
        "phrases": \[  
          "I'm not smart like you",  
          "You know I'm bad at this stuff",  
          "I never graduated, what do you expect",  
          "Just tell me what to do"  
        \]  
      }  
    }  
  },  
  "control\_destabilization": {  
    "version": "1.0",  
    "description": "Control and destabilization tactics",  
    "categories": {  
      "stonewalling": {  
        "pre\_stonewall": \[  
          "I'm done",  
          "I can't talk to you anymore",  
          "Don't contact me",  
          "This conversation is over",  
          "Whatever"  
        \],  
        "metadata\_tracking": \[  
          "communication\_gap\_duration",  
          "frequency\_analysis"  
        \]  
      },  
      "reactive\_abuse\_cycle": {  
        "provocation\_step": {  
          "description": "The Poke \- deliberate button pushing",  
          "triggers": \[  
            "character\_assassination",  
            "infidelity\_accusations",  
            "feigned\_incompetence"  
          \]  
        },  
        "reaction\_step": {  
          "description": "Target's emotional response",  
          "indicators": \[  
            "angry",  
            "upset",  
            "pissed",  
            "stop",  
            "why",  
            "exclamation\_points",  
            "all\_caps"  
          \]  
        },  
        "gotcha\_step": {  
          "description": "Weaponizing the reaction",  
          "phrases": \[  
            "See, you're losing it",  
            "This is what I have to deal with",  
            "You're the crazy one",  
            "I'm going to record this",  
            "Look at you, you're having an episode"  
          \]  
        }  
      },  
      "last\_minute\_changes": {  
        "keywords": \[  
          "change of plans",  
          "can't make it",  
          "something came up",  
          "have to cancel",  
          "not going to work"  
        \],  
        "phrases": \[  
          "We're not doing that anymore",  
          "I decided to do \[X\] instead",  
          "You'll have to figure it out"  
        \]  
      }  
    }  
  },  
  "sexual\_emotional\_weaponization": {  
    "version": "1.0",  
    "description": "Sexual and emotional manipulation patterns",  
    "categories": {  
      "love\_bombing": {  
        "keywords": \[  
          "perfect",  
          "amazing",  
          "soulmate",  
          "can't live without you",  
          "always",  
          "forever",  
          "everything",  
          "desperate",  
          "need you"  
        \],  
        "phrases": \[  
          "You're the only one who understands me",  
          "I've never felt this way before",  
          "I want to give you everything"  
        \]  
      },  
      "sexual\_shaming": {  
        "keywords": \[  
          "slut",  
          "whore",  
          "pervert",  
          "disgusting",  
          "sick",  
          "nasty",  
          "freak",  
          "used",  
          "cheap"  
        \],  
        "phrases": \[  
          "You're just like \[previous partner\]",  
          "No wonder everyone leaves you",  
          "Talking about what you enjoy makes you sound sick",  
          "To think I ever did \[X\] with you"  
        \]  
      }  
    }  
  },  
  "parental\_alienation": {  
    "version": "1.0",  
    "description": "Child-focused manipulation patterns",  
    "categories": {  
      "classic\_alienation": {  
        "phrases": \[  
          "\[Child's name\] doesn't want to see you",  
          "I have to protect the children from you"  
        \],  
        "tactics": \[  
          "sharing\_inappropriate\_info",  
          "first\_name\_references",  
          "fabricated\_quotes"  
        \]  
      },  
      "autism\_weaponization": {  
        "phrases": \[  
          "You can't handle his autism",  
          "interfering\_with\_medical\_care"  
        \],  
        "medical\_interference": \[  
          "appointment\_blocking",  
          "treatment\_undermining"  
        \]  
      }  
    }  
  },  
  "linguistic\_markers": {  
    "version": "1.0",  
    "description": "LIWC-style linguistic analysis markers",  
    "categories": {  
      "pronoun\_usage": {  
        "first\_person\_singular": \[  
          "I",  
          "me",  
          "my",  
          "mine",  
          "myself"  
        \],  
        "second\_person": \[  
          "you",  
          "your",  
          "yours",  
          "yourself"  
        \],  
        "analysis\_flags": \[  
          "high\_i\_talk",  
          "blame\_projection"  
        \]  
      },  
      "certainty\_absolutes": {  
        "keywords": \[  
          "always",  
          "never",  
          "nothing",  
          "everything",  
          "fact",  
          "obviously",  
          "clearly",  
          "literally",  
          "completely",  
          "impossible"  
        \]  
      },  
      "emotional\_dysregulation": {  
        "manic\_indicators": \[  
          "brilliant idea",  
          "I can solve everything",  
          "pressured\_speech",  
          "grandiosity"  
        \],  
        "depressive\_indicators": \[  
          "What's the point",  
          "It will never get better",  
          "I'm a failure",  
          "I can't do anything right"  
        \]  
      }  
    }  
  }  
}  
{  
  "patterns": {  
    "gaslighting": \[  
      "i never said that", "you imagined", "you’re imagining", "you are imagining",  
      "you’re paranoid", "that never happened", "no one will believe", "you’re crazy",  
      "you are crazy", "you’re just high", "just kidding", "you’re overreacting",  
      "this is the drugs talking"  
    \],  
    "blame\_shifting": \[  
      "this is your fault", "you made me", "because of you", "you started this",  
      "you always do this", "if you hadn’t", "look what you made me do"  
    \],  
    "minimizing": \[  
      "not a big deal", "you’re too sensitive", "calm down", "you’re being dramatic",  
      "get over it", "stop making a scene", "it was just a joke", "relax"  
    \],  
    "circular": \[  
      "what even is the point", "that’s not the point", "you keep changing",  
      "you know what i mean", "anyway", "whatever", "we’re not in high school"  
    \],  
    "contradiction": {  
      "denials": \["i don’t", "i do not", "i never", "i would never", "that’s not me", "i don’t use"\],  
      "platforms": \["snapchat", "snap", "instagram", "facebook", "tiktok"\]  
    }  
  },  
  "auxiliary": {  
    "substance\_alcohol": \["drink","drank","drunk","buzzed","tipsy","wasted","bottle","wine","beer","liquor","vodka","tequila","hungover","fireball"\],  
    "substance\_against\_you": \["crackhead","tweaker","addict","junkie","user","you’re just high","are you on something","this is the drugs talking"\],  
    "adderall\_control": \["adderall","addy","pills","script","share","split","your turn","how many did you take","i’m holding onto them for you","you can’t control yourself"\],  
    "infidelity\_places": \["huckleberry junction","huck’s","hucks"\],  
    "infidelity\_general": \["cheating","cheated","slept with","affair","secret","seeing someone","loyal","faithful","he’s just a friend","we just work together","you’re being jealous","why don’t you trust me"\],  
    "social\_platforms": \["snapchat","snap","instagram","facebook","tiktok"\],  
    "financial\_domestic": \["work","hours","job","tired","exhausted","broke","bills","rent","money","pay","afford","contribute","clean","cook","laundry","dishes","groceries","errands","lunch"\],  
    "financial\_weaponized": \["you don’t do anything","i’m the one who works hard","what do i get out of this","it’s your responsibility to provide"\],  
    "love\_bombing": \["perfect","amazing","soulmate","can’t live without you","always","forever","everything","desperate","need you","you’re the only one who understands me","i’ve never felt this way before","i want to give you everything"\],  
    "sexual\_shaming": \["slut","whore","pervert","disgusting","sick","nasty","freak","used","cheap","no wonder everyone leaves you","to think i ever did"\]  
  }  
}  
Regex tips (optional, only if you implement pre‑matching):

Case‑insensitive, word‑boundary where safe: (?i)\\blook what you made me do\\b

Slur variants (compact, careful): (?i)f\[a@\*\]g+(\[o0\])t+ (use sparingly; prefer exact quotes already present)

Platform denial pair detector: previously mentions \\b(snap|snapchat)\\b AND later (?i)\\b(i don’t|i never)\\s+(use|have)\\s+snap(chat)?\\b

5\) Starter Tag Cloud (unchanged core; aligns with your app)  
Use these tags in your UI/Search (not in the report output):

By Pattern: \#pattern:gaslighting, \#pattern:blame-shifting, \#pattern:minimizing, \#pattern:circular-argument, \#pattern:love-bombing, \#pattern:devaluation, \#pattern:hoovering

By Theme: \#theme:finances, \#theme:parenting, \#theme:isolation, \#theme:infidelity, \#theme:future-faking, \#theme:health

By Evidence Type: \#evidence:contradiction, \#evidence:admission, \#evidence:lie, \#evidence:key-quote

By Significance: \#event:major, \#event:minor, \#seed  
