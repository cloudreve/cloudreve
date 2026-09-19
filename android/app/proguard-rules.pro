# Keep kotlinx.serialization generated serializers
-keepclassmembers class org.cloudreve.android.api.** {
    *** Companion;
}
-keepclasseswithmembers class org.cloudreve.android.api.** {
    kotlinx.serialization.KSerializer serializer*(...);
}
-keepattributes *Annotation*, InnerClasses
-dontnote kotlinx.serialization.**
-dontwarn okhttp3.**
-dontwarn okio.**
