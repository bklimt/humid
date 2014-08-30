Humid
=====

A go app for controlling Philips Hue lightbulbs with a MIDI controller.
Specifically designed for use on the Raspberry Pi.

# Setup

Depends on my [MIDI package](https://github.com/bklimt/midi) and my [Hue package](https://github.com/bklimt/hue), so make sure to install the dependencies for those. Specifically, you'll need to make sure you install the ALSA asound development libraries.

You'll need to make sure your Hue is configured correctly. Build the `hue` command-line tool.

    go install github.com/bklimt/hue/...

Then press the link button on your Hue router and use the command-line tool to register the app.

    $GOPATH/bin/hue --hue_ip="YOUR.ROUTER.IP.ADDRESS" --register

Once that it complete, you can start the `humid` app to listen for MIDI events to control the lights.

    $GOPATH/bin/humid --hue_ip="YOUR.ROUTER.IP.ADDRESS"

While `humid` is running, you can see it as a MIDI receiver on your system by running:

    aconnect -loi

You'll need to use `aconnect` to bind the MIDI controller you want to use to `humid`, something like:

    aconnect 20 128

To change how the controller inputs map to lighting values, edit `presets.json`.
