// Calibration policy: stop at this weighted mean-square residual, and use a
// smaller update gain near a flat objective. These values are tuning knobs,
// not mathematical coefficients in the squared-error derivative.
const CONVERGENCE_LOSS_TOLERANCE: f64 = 0.000_5;
const FAST_STEP_GRADIENT_THRESHOLD: f64 = 0.125;
const FAST_STEP_GAIN: f64 = 0.25;
const FINE_STEP_GAIN: f64 = 0.05;

#[derive(Clone, Copy, Debug)]
pub struct Observation {
    pub measured: f64,
    pub expected: f64,
    pub weight: f64,
}

#[derive(Clone, Copy, Debug)]
pub struct SolverConfig {
    pub initial_offset: f64,
    pub max_iterations: usize,
}

#[derive(Clone, Copy, Debug, PartialEq)]
pub enum StopReason {
    Converged,
    IterationLimit,
}

#[derive(Clone, Copy, Debug)]
pub struct Calibration {
    pub offset: f64,
    pub mean_squared_error: f64,
    pub iterations: usize,
    pub reason: StopReason,
}

#[derive(Clone, Copy, Debug, PartialEq)]
pub enum FitError {
    NoObservations,
    InvalidObservation,
    InvalidInitialOffset,
}

fn validate(observations: &[Observation], initial: f64) -> Result<(), FitError> {
    if observations.is_empty() {
        return Err(FitError::NoObservations);
    }
    if !initial.is_finite() {
        return Err(FitError::InvalidInitialOffset);
    }
    if observations.iter().any(|sample| {
        !sample.measured.is_finite()
            || !sample.expected.is_finite()
            || !sample.weight.is_finite()
            || sample.weight <= 0.0
    }) {
        return Err(FitError::InvalidObservation);
    }
    Ok(())
}

fn objective(observations: &[Observation], offset: f64) -> (f64, f64) {
    let mut loss = 0.0;
    let mut gradient = 0.0;
    let mut total_weight = 0.0;
    for sample in observations {
        let residual = sample.measured + offset - sample.expected;
        loss += sample.weight * residual * residual;
        gradient += 2.0 * sample.weight * residual;
        total_weight += sample.weight;
    }
    (loss / total_weight, gradient / total_weight)
}

pub fn fit_offset(
    observations: &[Observation],
    config: SolverConfig,
) -> Result<Calibration, FitError> {
    validate(observations, config.initial_offset)?;
    let mut offset = config.initial_offset;
    for iteration in 0..config.max_iterations {
        let (loss, gradient) = objective(observations, offset);
        if loss <= CONVERGENCE_LOSS_TOLERANCE {
            return Ok(Calibration {
                offset,
                mean_squared_error: loss,
                iterations: iteration,
                reason: StopReason::Converged,
            });
        }
        let gain = if gradient.abs() > FAST_STEP_GRADIENT_THRESHOLD {
            FAST_STEP_GAIN
        } else {
            FINE_STEP_GAIN
        };
        offset -= gain * gradient;
    }
    let (loss, _) = objective(observations, offset);
    Ok(Calibration {
        offset,
        mean_squared_error: loss,
        iterations: config.max_iterations,
        reason: StopReason::IterationLimit,
    })
}
