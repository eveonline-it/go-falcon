Enhanced commit workflow with automatic changelog integration:

1. Check if the user provided --no-changelog flag
   - If yes: create commit without changelog prompt
   - If no: proceed to changelog workflow

2. After creating the commit, ask the user:
   "What type of change is this for the changelog?"
   
   Present options:
   1. added (new features)
   2. changed (existing functionality changes)  
   3. fixed (bug fixes)
   4. security (security improvements)
   5. deprecated (soon-to-be removed)
   6. removed (removed features)
   7. skip (don't update changelog)

3. Based on their choice, update the changelog file using the changelog script

4. Create concise git commits without Co-Authored-By and Generated lines

5. If user chooses "skip" or uses --no-changelog, only create the commit

Follow the conventional commit format but keep messages concise and clear.