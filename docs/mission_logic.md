
# Mission Logic

## Stage States

The 6 possible states for missions are:

- **ready**: Initial state for all stages when a mission is created.
- **started**: Stage is in progress.
- **finished**: Stage has been completed.
- **failed**: A fatal error occurred when the stage was running. Failed stages can be retried by starting them again.
- **excluded**: Stage won't be run and any stages that are dependent on this stage (downstream of this stage) will also be excluded. 
- **skipped**: Stage won't run, but the mission will continue as if this stage doesn't exist.

A stage can only be started if every upstream stage is finished, or if the 'ignore dependencies' flag is set to true in the request.  

The below table shows the result when attempting to change the state of a stage. The left most column gives the current
state of the stage.

|              | set to started | set to finished | set to failed | set to excluded | set to skipped |
|--------------|----------------|-----------------|---------------|-----------------|----------------|
| **ready**    | --> started    | _ERROR_         | _ERROR_       | --> excluded    | --> skipped    |
| **started**  | _ERROR_        | --> finished    | --> failed    | _ERROR_         | _ERROR_        |
| **finished** | _ERROR_        | _ERROR_         | _ERROR_       | (no change)     | (no change)    |
| **failed**   | --> started    | _ERROR_         | _ERROR_       | _ERROR_         | _ERROR_        |
| **excluded** | _ERROR_        | _ERROR_         | _ERROR_       | (no change)     | (no change)    |
| **skipped**  | _ERROR_        | _ERROR_         | _ERROR_       | (no change)     | (no change)    |

Stages can't be set back to 'ready' once they've changed state.
