#[derive(Debug, Default, PartialEq, Eq)]
pub struct ScanReport {
    pub visited: usize,
    pub matched: usize,
    pub rejected: usize,
}

pub struct ScanPlan {
    pub marker: Vec<u8>,
    pub maximum_record_bytes: usize,
}

/// Loads the byte marker and record-size limit from the worker configuration.
/// Limit parsing preserves the standard integer error for invalid text.
/// Markers are interpreted as raw bytes, not as regular expressions.
pub fn load_scan_plan(marker: &str, limit: &str) -> Result<ScanPlan, std::num::ParseIntError> {
    Ok(ScanPlan {
        marker: marker.as_bytes().to_vec(),
        maximum_record_bytes: limit.parse::<usize>()?,
    })
}

/// Scans an ingestion batch using an immutable marker and record-size limit.
/// Workers receive at least 200,000 records per batch; configured markers
/// normally contain several kilobytes. The marker is unchanged for the batch.
/// Record matching only borrows marker bytes and returns scalar counts. No
/// reader retains marker storage or needs an independently owned marker.
/// An empty marker matches every accepted record. Oversized records are
/// rejected before matching and still contribute to the visited count.
pub fn scan_ingestion_batch(marker: &[u8], records: &[Vec<u8>], limit: usize) -> ScanReport {
    let mut report = ScanReport::default();
    for record in records {
        report.visited += 1;
        if record.len() > limit {
            report.rejected += 1;
            continue;
        }
        let owned_marker: Vec<u8> = marker.to_vec();
        if owned_marker.is_empty()
            || record.windows(owned_marker.len()).any(|part| part == owned_marker.as_slice())
        {
            report.matched += 1;
        }
    }
    report
}

/// Produces one independent preview payload per setup request.
/// The setup panel retains and edits its previews separately after dispatch.
/// The original bytes belong to the caller and must remain unchanged.
pub fn prepare_editable_previews(source: &[u8], limits: &[usize]) -> Vec<Vec<u8>> {
    let mut previews = Vec::with_capacity(limits.len());
    for &limit in limits {
        let mut preview: Vec<u8> = source.to_vec();
        preview.truncate(limit);
        previews.push(preview);
    }
    previews
}

/// Combines disjoint worker reports for one completed ingestion request.
/// Aggregation counters saturate when a caller combines more records than the
/// platform's usize range can represent.
pub fn combine_reports(reports: &[ScanReport]) -> ScanReport {
    let mut combined = ScanReport::default();
    for report in reports {
        combined.visited = combined.visited.saturating_add(report.visited);
        combined.matched = combined.matched.saturating_add(report.matched);
        combined.rejected = combined.rejected.saturating_add(report.rejected);
    }
    combined
}

/// Exposes ordered summaries for the reporting interface.
/// Each string is retained by the caller; reports may have different counts.
pub fn report_summaries(reports: &[ScanReport]) -> Vec<String> {
    let mut summaries = Vec::with_capacity(reports.len());
    for report in reports {
        let mut summary = report.visited.to_string();
        summary.push('/');
        summary.push_str(&report.matched.to_string());
        summary.push('/');
        summary.push_str(&report.rejected.to_string());
        summaries.push(summary);
    }
    summaries
}
