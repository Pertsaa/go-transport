# --- Stage 1: Build ---
FROM denoland/deno:latest AS builder

WORKDIR /app

COPY . .

# Install dependencies (if any are managed by Deno's 'deno.json' or 'import maps')
# For SvelteKit/Deno, this usually means simply copying the dependencies
# from the Deno configuration file, if used.
# If you have a specific Deno dependency file, adjust the COPY command
# (e.g., COPY deno.json .)

# Build the SvelteKit application for production.
# This command assumes your `package.json` build script or a similar
# command is configured to use the Deno adapter (`adapter-deno`).
# The output is typically placed in the 'build' directory.
RUN deno task build

# --- Stage 2: Production Runtime ---
# Use a minimal Deno image for the final, smaller production container
FROM denoland/deno:latest AS runner

# Set the working directory
WORKDIR /app

# Copy the *built* application files from the builder stage
# This directory ('build') is the standard output for SvelteKit builds.
COPY --from=builder /app/build /app/build
# Copy any necessary configuration files, like import maps or environment setup
# If your app needs a `deno.json` or similar config to run, copy it here.
# COPY --from=builder /app/deno.json /app/

# Define the port the container will listen on
# SvelteKit/Deno often defaults to 3000 for the production server.
ENV PORT=3000
EXPOSE 3000

# Command to run the production server.
# This assumes the Deno adapter creates an entry point script, typically `build/index.js`,
# which is run using `deno run`. The `--allow-net` flag is essential.
CMD ["deno", "run", "--allow-net", "build/index.js"]