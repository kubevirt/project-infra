# Periodic Job Spreading Algorithm

## Problem Statement

When multiple periodic jobs are scheduled with arbitrary cron expressions, they can cluster at specific times, causing:

- **Resource contention**: Multiple jobs competing for the same compute resources
- **Queue buildup**: Jobs waiting for available resources
- **Unpredictable load**: High peaks and low troughs instead of steady load
- **Inefficient resource utilization**: Resources idle during low periods

### Real-World Example

In the kubevirt/kubevirt project, we had 23 periodic e2e test jobs with uneven cron schedules:

**Before spreading:**
- Peak: 17 jobs running concurrently
- 9 out of 48 thirty-minute windows had ≥12 concurrent jobs
- Uneven distribution: some windows had 17 jobs, others had 3-4

**After spreading:**
- Peak: 11 jobs running concurrently
- All 48 thirty-minute windows: 8-11 concurrent jobs
- Close to theoretical optimal: 9.6 concurrent jobs average

## Algorithm Overview

The spreading algorithm consists of four phases:

### Phase 1: Grouping by Frequency

Jobs are grouped based on how many times they run per day:

```
Frequency = Number of hours in cron expression
Period = 24 hours / Frequency
```

Examples:
- `cron: 30 0,6,12,18 * * *` → 4 times/day → 6-hour period
- `cron: 45 7,15,23 * * *` → 3 times/day → 8-hour period
- `cron: 20 4,16 * * *` → 2 times/day → 12-hour period

### Phase 2: Calculate Stagger Interval

For each frequency group, calculate the stagger interval:

```
Stagger Interval = Period / Number of Jobs in Group
```

This is the time offset between consecutive job starts.

Examples:
- 14 jobs at 4×/day: `360 minutes / 14 jobs = 25.7 minutes ≈ 25 minutes`
- 3 jobs at 3×/day: `480 minutes / 3 jobs = 160 minutes`
- 4 jobs at 2×/day: `720 minutes / 4 jobs = 180 minutes`

### Phase 3: Assign Start Times

Jobs are sorted alphabetically by name for deterministic assignment. Each job gets a start time offset:

```
Job[i].StartOffset = i * StaggerInterval
Job[i].Minute = StartOffset % 60
Job[i].StartHour = StartOffset / 60
```

The job then runs at intervals of `Period` hours:

```
Job[i].Hours = [StartHour, StartHour + Period, StartHour + 2*Period, ...]
```

### Phase 4: Update Cron Expressions

Convert the calculated times back to cron format:

```
cron: <Minute> <Hour1>,<Hour2>,... * * *
```

## Mathematical Properties

### Load Distribution

Given:
- `N` = total number of jobs
- `F` = frequency (times per day)
- `T` = total execution time (24 hours)

The algorithm guarantees:
- **Even spacing within groups**: Jobs in the same frequency group are evenly spaced
- **Minimal clustering**: No two jobs in the same group start at the same time
- **Bounded concurrency**: Peak concurrent jobs is minimized

### Optimal Stagger

The stagger interval is optimal when:

```
Stagger = Period / Jobs_in_Group
```

This creates the maximum possible separation between jobs while maintaining their required frequency.

### Load Flatness Metric

Define load flatness as the ratio of peak to average concurrent jobs:

```
Flatness = Peak_Concurrent / Average_Concurrent
```

Ideal flatness = 1.0 (perfect even distribution)

Before spreading: Flatness = 17 / 9.6 = 1.77
After spreading: Flatness = 11 / 9.6 = 1.15

## Edge Cases and Limitations

### 1. Non-Divisible Periods

When `Period` doesn't divide evenly by the number of jobs:

```
14 jobs × 360 minutes period = 25.71 minutes per job
```

We round down to 25 minutes, which creates slight unevenness but is unavoidable.

**Impact**: Last job might have slightly longer gap to first job than others.

### 2. Multiple Frequency Groups

Jobs with different frequencies have different periods. The algorithm handles each group independently.

**Example**: If you have 14 jobs at 4×/day and 3 jobs at 3×/day, they're spread independently:
- 4×/day jobs: spread every 25 minutes starting at 0:00
- 3×/day jobs: spread every 160 minutes starting at 0:00

**Potential issue**: Jobs from different groups might still overlap.

### 3. Fixed Time Requirements

Some jobs must run at specific times (e.g., "end of business day"). This tool doesn't handle such constraints.

**Workaround**: Exclude these jobs from spreading by using a more specific pattern.

### 4. Job Duration Variance

The algorithm assumes jobs start at staggered times but doesn't account for varying durations.

**Example**: If jobs take 30-90 minutes to complete, there will still be overlap.

**Solution**: This is fundamental to parallel execution. The goal is to minimize start-time clustering, not eliminate overlap entirely.

### 5. Alphabetical Ordering

Jobs are assigned times based on alphabetical order of their names.

**Impact**: Changing job names will change their assigned times.

**Rationale**: Provides deterministic, reproducible results.

## Examples

### Example 1: 4 Jobs at 4×/day

```
Input jobs:
- job-a: cron: 10 3,7,15,23 * * *
- job-b: cron: 20 1,9,17,23 * * *
- job-c: cron: 30 0,6,12,18 * * *
- job-d: cron: 40 2,8,14,20 * * *

Algorithm:
- Period: 6 hours = 360 minutes
- Stagger: 360 / 4 = 90 minutes

Assignment:
- job-a (index 0): offset 0 → 0:00, 6:00, 12:00, 18:00
- job-b (index 1): offset 90 → 1:30, 7:30, 13:30, 19:30
- job-c (index 2): offset 180 → 3:00, 9:00, 15:00, 21:00
- job-d (index 3): offset 270 → 4:30, 10:30, 16:30, 22:30

Output:
- job-a: cron: 0 0,6,12,18 * * *
- job-b: cron: 30 1,7,13,19 * * *
- job-c: cron: 0 3,9,15,21 * * *
- job-d: cron: 30 4,10,16,22 * * *

Load distribution: 1 job every 90 minutes
Peak concurrent: depends on job duration
```

### Example 2: Mixed Frequencies

```
Input:
- 2 jobs at 4×/day (6-hour period)
- 2 jobs at 2×/day (12-hour period)

Group 1 (4×/day):
- Stagger: 360 / 2 = 180 minutes
- job-1a: 0:00, 6:00, 12:00, 18:00
- job-1b: 3:00, 9:00, 15:00, 21:00

Group 2 (2×/day):
- Stagger: 720 / 2 = 360 minutes
- job-2a: 0:00, 12:00
- job-2b: 6:00, 18:00

Timeline:
00:00: job-1a, job-2a (2 concurrent)
03:00: job-1b (1 concurrent)
06:00: job-1a, job-2b (2 concurrent)
09:00: job-1b (1 concurrent)
12:00: job-1a, job-2a (2 concurrent)
15:00: job-1b (1 concurrent)
18:00: job-1a, job-2b (2 concurrent)
21:00: job-1b (1 concurrent)

Result: Maximum 2 concurrent jobs, average 1.5
```

## Theoretical Bounds

### Lower Bound on Peak Concurrency

Given `N` jobs running `F` times per day with average duration `D` hours:

```
Average_Concurrent = (N × F × D) / 24

Peak_Concurrent ≥ Average_Concurrent
```

This is the theoretical minimum you cannot go below, regardless of spreading.

### Upper Bound on Stagger Interval

For jobs running `F` times per day:

```
Max_Stagger = 24 / F hours
```

You cannot stagger more than the period without changing the frequency.

### Optimal Spacing

For `N` jobs at frequency `F`:

```
Optimal_Stagger = (24 / F) / N hours
```

This is exactly what our algorithm implements.

## Comparison with Other Approaches

### 1. Random Distribution

**Approach**: Assign random start times

**Pros**: Simple, no coordination needed

**Cons**:
- Unpredictable clustering (birthday paradox)
- No guaranteed bounds on peak concurrency
- Non-deterministic (changes on each run)

### 2. Round-Robin

**Approach**: Assign jobs to time slots in sequence

**Pros**: Simple, deterministic

**Cons**:
- Doesn't account for different frequencies
- May create artificial frequency changes

### 3. Bin Packing

**Approach**: Try to pack jobs into time slots to minimize peak

**Pros**: Considers job duration

**Cons**:
- NP-hard problem
- Requires knowledge of job duration
- Complex to implement

### 4. This Algorithm (Frequency-Aware Spreading)

**Pros**:
- Preserves job frequencies
- Deterministic and reproducible
- Simple to understand and implement
- Provides mathematical guarantees
- Works with incomplete information (doesn't need duration)

**Cons**:
- Doesn't account for job duration
- Doesn't handle time-of-day constraints
- Alphabetical ordering might not be ideal for all use cases

## Future Enhancements

Potential improvements to the algorithm:

1. **Duration-Aware Spreading**: Account for actual job durations when spreading
2. **Constraint-Based**: Allow jobs to specify time windows or constraints
3. **Multi-Objective Optimization**: Balance load, resource types, priority
4. **Adaptive Learning**: Learn from past executions to optimize future schedules
5. **Resource-Aware**: Consider different resource pools and constraints
