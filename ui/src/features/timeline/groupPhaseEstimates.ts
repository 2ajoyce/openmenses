import type { PhaseEstimate } from "@gen/openmenses/v1/model_pb";

/**
 * Groups consecutive same-phase PhaseEstimate records into date ranges.
 * This prevents noisy repetition in the timeline (e.g., 14 separate cards for a 2-week phase).
 * Returns a list of grouped PhaseEstimate arrays, where each array contains
 * consecutive same-phase estimates.
 */
export function groupPhaseEstimates(
  estimates: PhaseEstimate[],
): PhaseEstimate[][] {
  if (estimates.length === 0) {
    return [];
  }

  const groups: PhaseEstimate[][] = [];
  let currentGroup: PhaseEstimate[] = [estimates[0]!];

  for (let i = 1; i < estimates.length; i++) {
    const current = estimates[i]!;
    const previous = estimates[i - 1]!;

    // If phase is the same, add to current group
    if (current.phase === previous.phase) {
      currentGroup.push(current);
    } else {
      // Phase changed, start a new group
      groups.push(currentGroup);
      currentGroup = [current];
    }
  }

  // Add the last group
  if (currentGroup.length > 0) {
    groups.push(currentGroup);
  }

  return groups;
}
