# Server Deployment Fix for Bridge Pairing

## Issue

The production server at `https://repset.onezy.in/api/v1/devices/pair` is returning HTTP 500 errors because the `generateDeviceCredentials` function is missing.

## Solution

### 1. Add Missing Function

Copy the `device-auth.ts` file to your server project's `lib/` directory:

```bash
# In your server repository
cp device-auth.ts lib/device-auth.ts
```

### 2. Update Pairing API Route

Ensure your pairing API route imports and uses the function:

```typescript
// app/api/v1/devices/pair/route.ts
import { generateDeviceCredentials } from "@/lib/device-auth";

export async function POST(request: Request) {
  try {
    const { pairCode, deviceInfo } = await request.json();

    // Validate pair code exists and is not expired
    const pairCodeRecord = await prisma.pairCode.findFirst({
      where: {
        code: pairCode,
        expiresAt: { gt: new Date() },
        usedAt: null,
      },
      include: { gym: true },
    });

    if (!pairCodeRecord) {
      return Response.json(
        { error: "Invalid or expired pair code" },
        { status: 400 }
      );
    }

    // Generate device credentials
    const { deviceId, deviceKey } = generateDeviceCredentials(
      pairCodeRecord.gymId
    );

    // Create bridge deployment record
    const bridgeDeployment = await prisma.bridgeDeployment.create({
      data: {
        deviceId,
        deviceKey,
        gymId: pairCodeRecord.gymId,
        hostname: deviceInfo.hostname,
        platform: deviceInfo.platform,
        version: deviceInfo.version,
        tier: deviceInfo.tier || "normal",
        status: "active",
      },
    });

    // Mark pair code as used
    await prisma.pairCode.update({
      where: { id: pairCodeRecord.id },
      data: { usedAt: new Date() },
    });

    return Response.json({
      deviceId,
      deviceKey,
      config: {
        heartbeatInterval: 60,
        queueMaxSize: 10000,
        unlockDuration: 3000,
      },
    });
  } catch (error) {
    console.error("Pairing error:", error);
    return Response.json({ error: "Internal server error" }, { status: 500 });
  }
}
```

### 3. Deploy to Production

After adding the missing function:

```bash
# Deploy to your hosting platform (Vercel, Railway, etc.)
git add .
git commit -m "Fix: Add missing generateDeviceCredentials function for bridge pairing"
git push origin main

# Or deploy directly
vercel --prod
# or
railway up
```

### 4. Test the Fix

Once deployed, test the pairing endpoint:

```bash
curl -X POST https://repset.onezy.in/api/v1/devices/pair \
  -H "Content-Type: application/json" \
  -d '{
    "pairCode": "TEST-CODE-HERE",
    "deviceInfo": {
      "hostname": "test-host",
      "platform": "windows",
      "version": "1.0.0",
      "tier": "normal"
    }
  }'
```

## Expected Result

After deployment, the bridge pairing should work:

- ✅ HTTP 200 response with device credentials
- ✅ Bridge can pair successfully
- ✅ Service starts without CGO errors
- ✅ Complete installation and pairing process works smoothly

## Files to Deploy

1. `lib/device-auth.ts` - The missing function
2. Updated pairing API route (if needed)

## Verification

Test with the bridge installation:

```powershell
$pairCode = "YOUR_FRESH_PAIR_CODE"
$script = iwr -useb https://raw.githubusercontent.com/MAnishSingh13275/repset_bridge/main/scripts/install-bridge.ps1
Invoke-Expression "& { $($script.Content) } -PairCode '$pairCode'"
```

Should complete successfully without HTTP 500 errors.
