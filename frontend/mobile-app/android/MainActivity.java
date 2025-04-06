package com.iaros.mobileapp;

import android.os.Bundle;
import androidx.appcompat.app.AppCompatActivity;

public class MainActivity extends AppCompatActivity {
    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        // Set content view and initialize crash reporting.
        setContentView(R.layout.activity_main);
        // Inline: Crash reporting integration using Firebase Crashlytics.
    }
}
